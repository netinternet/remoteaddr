package remoteaddr_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/netinternet/remoteaddr"
	"github.com/stretchr/testify/require"
)

func TestIP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testObj := remoteaddr.Parse()
		testObj.AddForwarders([]string{"8.8.8.0/24"})
		testObj.AddHeaders([]string{"True-Client-IP"})
		ip, port := testObj.IP(r)

		// Test
		require.Contains(t, testObj.Forwarders, "8.8.8.0/24")
		require.Contains(t, testObj.Headers, "True-Client-IP")
		require.Equal(t, "127.0.0.1", ip)
		require.NotZero(t, port)
	}))
	defer ts.Close()

	_, err := http.Get(ts.URL)
	require.NoError(t, err)
}

func newTestParser() *remoteaddr.Addr {
	parser := remoteaddr.Parse()
	parser.AddForwarders([]string{"127.0.0.0/8"})
	return parser
}
func TestXForwardedForSpoofing(t *testing.T) {
	realClientIP := "85.100.50.25"
	spoofedIP := "31.69.99.22"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parser := newTestParser()
		ip, _ := parser.IP(r)

		require.Equal(t, realClientIP, ip, "X-Forwarded-For spoofing bypass edilmemeli")
		require.NotEqual(t, spoofedIP, ip, "Sahte IP kabul edilmemeli")
	}))
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL, nil)
	require.NoError(t, err)

	// Saldırgan sahte IP gönderir, Cloudflare gerçek IP'yi sona ekler
	req.Header.Set("X-Forwarded-For", spoofedIP+", "+realClientIP)
	req.Header.Set("CF-Connecting-IP", realClientIP)

	_, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
}
func TestCFConnectingIPPriority(t *testing.T) {
	realClientIP := "85.100.50.25"
	spoofedIP := "31.69.99.22"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parser := newTestParser()
		ip, _ := parser.IP(r)

		require.Equal(t, realClientIP, ip, "CF-Connecting-IP öncelikli olmalı")
	}))
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL, nil)
	require.NoError(t, err)

	req.Header.Set("CF-Connecting-IP", realClientIP)
	req.Header.Set("X-Forwarded-For", spoofedIP)

	_, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
}

func TestXForwardedForRightToLeft(t *testing.T) {
	realClientIP := "85.100.50.25"
	spoofedIP := "31.69.99.22"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parser := newTestParser()
		ip, _ := parser.IP(r)

		require.Equal(t, realClientIP, ip, "X-Forwarded-For sağdan sola parse edilmeli")
	}))
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL, nil)
	require.NoError(t, err)

	// CF-Connecting-IP yok, sadece X-Forwarded-For
	// Saldırgan soldan sahte IP ekler, proxy sağdan gerçek IP ekler
	req.Header.Set("X-Forwarded-For", spoofedIP+", "+realClientIP)

	_, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
}
func TestDirectConnectionIgnoresHeaders(t *testing.T) {
	spoofedIP := "31.69.99.22"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Forwarder listesini boşalt - sadece doğrudan bağlantı test et
		parser := &remoteaddr.Addr{
			Forwarders: []string{},
			Headers:    []string{"CF-Connecting-IP", "X-Forwarded-For", "X-Real-Ip"},
		}
		ip, _ := parser.IP(r)

		// RemoteAddr kullanılmalı (127.0.0.1), header'lar değil
		require.Equal(t, "127.0.0.1", ip, "Doğrudan bağlantıda header'lar yok sayılmalı")
		require.NotEqual(t, spoofedIP, ip)
	}))
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL, nil)
	require.NoError(t, err)

	req.Header.Set("X-Forwarded-For", spoofedIP)
	req.Header.Set("CF-Connecting-IP", spoofedIP)

	_, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
}
func TestMultiProxyChain(t *testing.T) {
	realClientIP := "85.100.50.25"
	spoofedIP := "31.69.99.22"
	cloudflareIP := "103.21.244.10" // Cloudflare IP aralığında

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parser := newTestParser()
		ip, _ := parser.IP(r)

		require.Equal(t, realClientIP, ip, "Proxy zincirinde gerçek client IP bulunmalı")
		require.NotEqual(t, spoofedIP, ip)
		require.NotEqual(t, cloudflareIP, ip)
	}))
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL, nil)
	require.NoError(t, err)

	// Zincir: spoofed, real_client, cloudflare_ip
	req.Header.Set("X-Forwarded-For", spoofedIP+", "+realClientIP+", "+cloudflareIP)
	req.Header.Set("CF-Connecting-IP", realClientIP)

	_, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
}
func TestSingleXForwardedForSpoofing(t *testing.T) {
	spoofedIP := "31.69.99.22"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parser := newTestParser()
		ip, _ := parser.IP(r)

		// Sadece sahte IP var ve non-forwarder, kabul edilecek
		// Çünkü proxy zincirinde başka IP yok
		// Bu durumda en sağdaki (ve tek) non-forwarder IP döner
		require.Equal(t, spoofedIP, ip)
	}))
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL, nil)
	require.NoError(t, err)

	// CF-Connecting-IP doğru IP'yi verir
	// Ama bu testte CF-Connecting-IP yok, sadece tek IP'li XFF
	req.Header.Set("X-Forwarded-For", spoofedIP)

	_, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
}
