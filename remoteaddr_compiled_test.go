package remoteaddr_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/netinternet/remoteaddr"
	"github.com/stretchr/testify/require"
)

func req(remote string, hdr map[string]string) *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = remote
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	return r
}

// The compiled fast path (Parse/AddForwarders) must return exactly the same
// result as a struct-literal parser with identical config, which exercises the
// legacy parse-per-call path.
func TestCompiledPathParityWithLegacy(t *testing.T) {
	extraFwd := []string{"198.51.100.0/24", "not-a-cidr"}
	extraHdr := []string{"X-Custom-IP"}

	fast := remoteaddr.Parse().AddForwarders(extraFwd).AddHeaders(extraHdr)
	legacy := &remoteaddr.Addr{
		Forwarders:  append(remoteaddr.Parse().Forwarders, extraFwd...),
		Headers:     append(remoteaddr.Parse().Headers, extraHdr...),
		PortHeaders: remoteaddr.Parse().PortHeaders,
	}

	cases := []*http.Request{
		req("203.0.113.7:4444", nil),
		req("[2001:db8::7]:4444", nil),
		req("10.1.0.8:1234", map[string]string{"X-Forwarded-For": "203.0.113.7"}),
		req("104.16.1.1:443", map[string]string{"CF-Connecting-IP": "203.0.113.9"}),
		req("104.16.1.1:443", map[string]string{"X-Forwarded-For": "198.51.100.1, 10.0.0.5, 203.0.113.9"}),
		req("10.1.0.8:1234", map[string]string{"X-Forwarded-For": "10.0.0.5, 172.16.3.3"}),
		req("10.1.0.8:1234", map[string]string{"X-Forwarded-For": "garbage, , 203.0.113.9"}),
		req("192.168.1.1:9999", nil),
		req("[2606:4700::1]:443", map[string]string{"CF-Connecting-IP": "203.0.113.9"}),
		req("198.51.100.7:80", map[string]string{"X-Forwarded-For": "203.0.113.9"}),
		req("10.1.0.8:80", map[string]string{"X-Custom-IP": "203.0.113.11"}),
		req("10.1.0.8:80", map[string]string{"X-Forwarded-For": "::ffff:203.0.113.9"}),
		req("10.1.0.8:80", map[string]string{"CF-Connecting-IP": "203.0.113.1", "True-Client-Port": "54321"}),
	}
	for _, r := range cases {
		fi, fp := fast.IP(r)
		li, lp := legacy.IP(r)
		require.Equal(t, li, fi, "remote=%s hdr=%v", r.RemoteAddr, r.Header)
		require.Equal(t, lp, fp, "remote=%s hdr=%v", r.RemoteAddr, r.Header)
	}
}

// Direct struct-literal construction (no Parse) must keep working via the
// legacy path — backward compatibility guard.
func TestStructLiteralStillWorks(t *testing.T) {
	p := &remoteaddr.Addr{
		Forwarders: []string{"127.0.0.0/8"},
		Headers:    []string{"X-Real-Ip"},
	}
	ip, port := p.IP(req("127.0.0.1:5555", map[string]string{"X-Real-Ip": "203.0.113.7"}))
	require.Equal(t, "203.0.113.7", ip)
	require.Equal(t, "-1", port)
}

// Two parsers extending the shared default set must not contaminate each other
// (append must reallocate instead of writing into the shared backing array).
func TestParsersDoNotShareForwarderBacking(t *testing.T) {
	a := remoteaddr.Parse().AddForwarders([]string{"198.51.100.0/24"})
	b := remoteaddr.Parse().AddForwarders([]string{"192.0.2.0/24"})

	r := req("198.51.100.7:80", map[string]string{"X-Forwarded-For": "203.0.113.9"})
	ipA, _ := a.IP(r)
	require.Equal(t, "203.0.113.9", ipA, "parser A kendi ek prefix'ine güvenmeli")
	ipB, _ := b.IP(r)
	require.Equal(t, "198.51.100.7", ipB, "parser B, A'nın prefix'ini görmemeli")
}

func BenchmarkIPDirect(b *testing.B) {
	r := req("203.0.113.7:4444", nil)
	p := remoteaddr.Parse()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = p.IP(r)
	}
}

func BenchmarkIPBehindCF(b *testing.B) {
	r := req("104.16.1.1:443", map[string]string{"CF-Connecting-IP": "203.0.113.7"})
	p := remoteaddr.Parse()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = p.IP(r)
	}
}

func BenchmarkParseAndIP(b *testing.B) {
	r := req("104.16.1.1:443", map[string]string{"CF-Connecting-IP": "203.0.113.7"})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = remoteaddr.Parse().AddForwarders([]string{"198.51.100.0/24"}).IP(r)
	}
}
