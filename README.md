# remoteaddr
Go http real ip header parser module

A forwarders such as a reverse proxy or Cloudflare find the real IP address from the requests made to the http server behind it. Local IP addresses and CloudFlare ip addresses are defined by default within the module. It is possible to define more forwarder IP addresses.

In Turkey, it is obligatory to keep the port information of IP addresses shared with cgnat by the law no 5651. For this reason, dst port information is also given along with the IP addresses. **If the IP address is behind a proxy, the dst port information is returned as `-1`** — the port on `r.RemoteAddr` belongs to the proxy's own connection, not the real client, so exposing it would be misleading. Some proxies forward the real client port in a header (Cloudflare `Cf-Connecting-Port`, Heimwall `True-Client-Port`); when a request comes from a trusted forwarder, that header is read and validated (must be a `1-65535` TCP port), otherwise the port stays `-1`. Additional port headers can be registered with `AddPortHeaders`.

## Usage

```
go get -u github.com/netinternet/remoteaddr
```

```go
// remoteaddr.Parse().IP(*http.Request) return to string IPv4 or IPv6 address
```

## Example

Run a simple web server and get the real IP address to string format

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/netinternet/remoteaddr"
)

func root(w http.ResponseWriter, r *http.Request) {
	ip, port := remoteaddr.Parse().IP(r)
	fmt.Fprintf(w, "Your IP address is "+ip+" and dst port "+port)
}

func main() {
	http.HandleFunc("/", root)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

```

## Example 2 (Nginx or another web service forwarder address)

**AddForwarders([]string{"8.8.8.0/24"})** = Add a new multiple forwarder prefixes

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/netinternet/remoteaddr"
)

func root(w http.ResponseWriter, r *http.Request) {
	ip, port := remoteaddr.Parse().AddForwarders([]string{"8.8.8.0/24"}).IP(r)
	fmt.Fprintf(w, "Your IP address is "+ip+" and dst port "+port)
}

func main() {
	http.HandleFunc("/", root)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

```

## Example 4 (Add an alternative header for real client port)

**AddPortHeaders([]string{"X-Real-Port"})** = Add a new multiple real client port headers (defaults already include `Cf-Connecting-Port` and `True-Client-Port`)

```go
func root(w http.ResponseWriter, r *http.Request) {
	ip, port := remoteaddr.Parse().AddPortHeaders([]string{"X-Real-Port"}).IP(r)
	fmt.Fprintf(w, "Your IP address is "+ip+" and dst port "+port)
}
```

## Example 3 (Add an alternative header for real IP address)

**AddHeaders([]string{"True-Client-IP"})** = Add a new multiple real ip headers

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/netinternet/remoteaddr"
)

func root(w http.ResponseWriter, r *http.Request) {
	ip, port := remoteaddr.Parse().AddHeaders([]string{"True-Client-IP"}).IP(r)
	fmt.Fprintf(w, "Your IP address is "+ip+" and dst port "+port)
}

func main() {
	http.HandleFunc("/", root)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

```

## Example 5 (Not behind Cloudflare? Replace the trusted forwarder list)

The built-in defaults trust RFC1918 private ranges **and the Cloudflare IP space**.
If your service is **not** actually behind Cloudflare, keeping those defaults widens
your header-spoofing surface: anyone can route a request through Cloudflare (a
Worker, any proxied zone) so that it arrives from a "trusted" CF address carrying a
forged `X-Forwarded-For`. Use `SetForwarders` to **replace** the whole list with
only the proxies you really operate behind:

```go
func root(w http.ResponseWriter, r *http.Request) {
	ip, port := remoteaddr.Parse().
		SetForwarders([]string{"10.0.0.0/8", "203.0.113.10/32"}). // only YOUR proxies
		SetHeaders([]string{"X-Real-Ip"}).                        // only the header YOUR proxy sets
		SetPortHeaders(nil).                                      // disable port headers entirely
		IP(r)
	fmt.Fprintf(w, "Your IP address is "+ip+" and dst port "+port)
}
```

`SetForwarders(nil)` trusts no forwarder at all: headers are always ignored and
`r.RemoteAddr` is returned as-is. `Set*` methods replace, `Add*` methods extend.

## Performance

Forwarder prefixes are compiled once (`net/netip`) when the parser is built;
`IP()` performs zero parsing per request (~79 ns, 0 allocs for a direct client on
Apple silicon — previously ~1.9 µs and 116 allocs when every call re-parsed each
CIDR). Build the parser once and reuse it if you can; even per-request
construction stays cheap because the default set is pre-compiled at package init.
