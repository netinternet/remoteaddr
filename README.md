# remoteaddr
Go http real ip header parser module

A forwarders such as a reverse proxy or Cloudflare find the real IP address from the requests made to the http server behind it. Local IP addresses and CloudFlare ip addresses are defined by default within the module. It is possible to define more forwarder IP addresses.

## Usage

```
go get -u github.com/netinternet/remoteaddr
```

```
// remoteaddr.Parse().IP(*http.Request) return to string IPv4 or IPv6 address
```

## Example

Run a simple web server and get the real IP address to string format

```
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/netinternet/remoteaddr"
)

func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Your IP address is "+remoteaddr.Parse().IP(r))
}

func main() {
	http.HandleFunc("/", root)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

```

## Example 2 (Nginx or another web service forwarder address)

**AddForwarders([]string{"8.8.8.0/24"})** = Add a new multiple forwarder prefixes

```
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/netinternet/remoteaddr"
)

func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Your IP address is "+remoteaddr.Parse().AddForwarders([]string{"8.8.8.0/24"}).IP(r))
}

func main() {
	http.HandleFunc("/", root)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

```

## Example 3 (Add an alternative header for real IP address)

**AddHeaders([]string{"True-Client-IP"})** = Add a new multiple real ip headers

```
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/netinternet/remoteaddr"
)

func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Your IP address is "+remoteaddr.Parse().AddHeaders([]string{"True-Client-IP"}).IP(r))
}

func main() {
	http.HandleFunc("/", root)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

```
