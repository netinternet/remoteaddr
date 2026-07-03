package remoteaddr

import (
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
)

type Addr struct {
	Forwarders  []string
	Headers     []string
	PortHeaders []string

	// compiled holds the pre-parsed CIDR networks for Forwarders (netip.Prefix:
	// value type, allocation-free Contains). It is built by Parse()/AddForwarders()
	// so the hot path (IP → prefix match) does not call net.ParseCIDR on every
	// request (~30 parses + allocations per call before). compiledN records how
	// many Forwarders entries were compiled; if the caller constructed Addr as a
	// struct literal or mutated Forwarders directly, the counts differ and IP()
	// falls back to the legacy parse-per-call path, keeping full backward
	// compatibility.
	compiled  []netip.Prefix
	compiledN int
}

// defaultCompiled caches the parsed default prefix set once per process, so even
// callers that build a new parser per request (Parse().IP(r)) pay no parse cost.
var (
	defaultForwarders []string
	defaultCompiled   []netip.Prefix
)

func init() {
	list := defaultPrefixList()
	// Full slice expressions pin cap == len: a later append on a parser's own
	// Forwarders/compiled slice must reallocate instead of writing into the
	// shared backing array (two parsers appending concurrently would otherwise
	// race and cross-contaminate each other's prefix lists).
	defaultForwarders = list[:len(list):len(list)]
	compiled := compileCIDRs(defaultForwarders)
	defaultCompiled = compiled[:len(compiled):len(compiled)]
}

// compileCIDRs parses a prefix list once. Invalid entries are skipped, matching
// the legacy path (which ignores ParseCIDR errors).
func compileCIDRs(list []string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(list))
	for _, s := range list {
		if p, err := netip.ParsePrefix(strings.TrimSpace(s)); err == nil {
			out = append(out, p.Masked())
		}
	}
	return out
}

// defaultPrefixList returns the built-in trusted forwarder prefixes.
func defaultPrefixList() []string {
	// RFC1918 IPv4 Private Address Space
	local_prefixes := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}
	waf_prefixes := []string{
		// CloudFlare IP Address Space; https://www.cloudflare.com/ips/
		"103.21.244.0/22",
		"103.22.200.0/22",
		"103.31.4.0/22",
		"104.16.0.0/13",
		"104.24.0.0/14",
		"108.162.192.0/18",
		"131.0.72.0/22",
		"141.101.64.0/18",
		"162.158.0.0/15",
		"172.64.0.0/13",
		"173.245.48.0/20",
		"188.114.96.0/20",
		"190.93.240.0/20",
		"197.234.240.0/22",
		"198.41.128.0/17",
		"2400:cb00::/32",
		"2606:4700::/32",
		"2803:f800::/32",
		"2405:b500::/32",
		"2405:8100::/32",
		"2a06:98c0::/29",
		"2c0f:f248::/32",

		// HEIMWALL IPs
		"159.253.42.0/24",
		"94.102.14.5/24",
		"2a03:2100:a::/48",
		"2a03:2100:b::/48",
	}
	return append(local_prefixes, waf_prefixes...)
}

// Inital function
func Parse() *Addr {
	return &Addr{
		Forwarders:  defaultForwarders,
		Headers:     []string{"CF-Connecting-IP", "X-Forwarded-For", "X-Real-Ip"},
		PortHeaders: []string{"Cf-Connecting-Port", "True-Client-Port"},
		// Pre-compiled default set is shared read-only across all parsers; both
		// slices are only ever appended-to via AddForwarders, which reallocates
		// (cap == len), so parsers can never write into the shared backing array.
		compiled:  defaultCompiled,
		compiledN: len(defaultForwarders),
	}
}

// Add more Forwarder Prefixes
func (a *Addr) AddForwarders(prefixes []string) *Addr {
	a.Forwarders = append(a.Forwarders, prefixes...)
	// Keep the compiled cache in sync so the hot path stays parse-free. If the
	// caller had mutated Forwarders directly beforehand, counts already differ
	// and isForwarders uses the legacy path — appending here keeps it that way.
	if a.compiledN == len(a.Forwarders)-len(prefixes) {
		a.compiled = append(a.compiled, compileCIDRs(prefixes)...)
		a.compiledN = len(a.Forwarders)
	}
	return a
}

// Add more Real IP Address Headers
func (a *Addr) AddHeaders(headers []string) *Addr {
	a.Headers = append(a.Headers, headers...)
	return a
}

// Add more Real Client Port Headers
func (a *Addr) AddPortHeaders(headers []string) *Addr {
	a.PortHeaders = append(a.PortHeaders, headers...)
	return a
}

// realPort reads the client port from a trusted forwarder's port header.
// Returns "-1" if no header carries a valid TCP port (1-65535).
func (a *Addr) realPort(r *http.Request) string {
	for _, h := range a.PortHeaders {
		value := strings.TrimSpace(r.Header.Get(h))
		if value == "" {
			continue
		}
		if p, err := strconv.Atoi(value); err == nil && p >= 1 && p <= 65535 {
			return value
		}
	}
	return "-1"
}

// isForwarderAddr reports whether ip belongs to a trusted forwarder prefix —
// the compiled fast path, zero allocations. Unmap keeps parity with net.IPNet:
// a v4-mapped IPv6 address ("::ffff:1.2.3.4") must match IPv4 prefixes. netip's
// Contains already returns false for zoned IPv6 addresses, matching the legacy
// behavior where net.ParseIP rejected them.
func (a *Addr) isForwarderAddr(ip netip.Addr) bool {
	ip = ip.Unmap()
	for _, p := range a.compiled {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// Helper function — legacy parse-per-call check, kept byte-for-byte for Addr
// values built as struct literals (compiled cache absent or stale).
func (a *Addr) isForwarders(ip net.IP) bool {
	for _, forwarder := range a.Forwarders {
		_, cidr, err := net.ParseCIDR(forwarder)
		if err != nil || cidr == nil {
			continue
		}
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// Add http.request to find real IPv4 or IPv6 address and destination port
func (a *Addr) IP(r *http.Request) (ipaddr string, port string) {
	ipaddr, port, _ = net.SplitHostPort(r.RemoteAddr)
	if a.compiledN == len(a.Forwarders) {
		return a.ipCompiled(r, ipaddr, port)
	}
	return a.ipLegacy(r, ipaddr, port)
}

// ipCompiled is the zero-parse fast path used by Parse()/AddForwarders() users.
func (a *Addr) ipCompiled(r *http.Request, ipaddr, port string) (string, string) {
	addr, err := netip.ParseAddr(ipaddr)
	if err != nil || !a.isForwarderAddr(addr) {
		return ipaddr, port
	}
	port = a.realPort(r)
	for _, h := range a.Headers {
		header := r.Header.Get(h)
		if header == "" {
			continue
		}
		// Walk comma-separated hops right-to-left without allocating a slice.
		for len(header) > 0 {
			var seg string
			if i := strings.LastIndexByte(header, ','); i >= 0 {
				seg, header = header[i+1:], header[:i]
			} else {
				seg, header = header, ""
			}
			realIP, perr := netip.ParseAddr(strings.TrimSpace(seg))
			if perr != nil {
				continue
			}
			if !a.isForwarderAddr(realIP) {
				return realIP.Unmap().String(), port
			}
		}
	}
	return ipaddr, port
}

// ipLegacy preserves the original behavior exactly for struct-literal parsers.
func (a *Addr) ipLegacy(r *http.Request, ipaddr, port string) (string, string) {
	if a.isForwarders(net.ParseIP(ipaddr)) {
		port = a.realPort(r)
		for _, h := range a.Headers {
			header := r.Header.Get(h)
			if header == "" {
				continue
			}
			ips := strings.Split(header, ",")
			for i := len(ips) - 1; i >= 0; i-- {
				realIP := net.ParseIP(strings.TrimSpace(ips[i]))
				if realIP == nil {
					continue
				}
				if !a.isForwarders(realIP) {
					ipaddr = realIP.String()
					return ipaddr, port
				}
			}
		}
	}
	return ipaddr, port
}
