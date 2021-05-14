package remoteaddr

import (
	"net"
	"net/http"
	"strings"
)

type Addr struct {
	Forwarders []string
	Headers    []string
}

// Inital function
func Parse() *Addr {
	// RFC1918 IPv4 Private Address Space
	local_prefixes := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}
	// CloudFlare IP Address Space; https://www.cloudflare.com/ips/
	cf_prefixes := []string{
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
	}
	return &Addr{
		Forwarders: append(local_prefixes, cf_prefixes...),
		Headers:    []string{"X-Forwarded-For", "X-Real-Ip", "CF-Connecting-IP"},
	}
}

// Add more Forwarder Prefixes
func (a *Addr) AddForwarders(prefixes []string) *Addr {
	a.Forwarders = append(a.Forwarders, prefixes...)
	return a
}

// Add more Real IP Address Headers
func (a *Addr) AddHeaders(headers []string) *Addr {
	a.Headers = append(a.Headers, headers...)
	return a
}

// Helper function
func (a *Addr) isForwarders(ip net.IP) bool {
	for _, forwarder := range a.Forwarders {
		if _, cidr, _ := net.ParseCIDR(forwarder); cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// Add http.request to find real IPv4 or IPv6 address
func (a *Addr) IP(r *http.Request) string {
	ipaddr, _, _ := net.SplitHostPort(r.RemoteAddr)
	if a.isForwarders(net.ParseIP(ipaddr)) {
		for _, h := range a.Headers {
			for _, ip := range strings.Split(r.Header.Get(h), ",") {
				realIP := net.ParseIP(strings.Replace(ip, " ", "", -1))
				if check := net.ParseIP(realIP.String()); check != nil {
					ipaddr = realIP.String()
					if !a.isForwarders(net.ParseIP(ipaddr)) {
						break
					}
				}
			}
		}
	}
	return ipaddr
}
