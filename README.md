# remoteaddr
Go real IP adress package

## Usage

```
go get -u github.com/netinternet/remoteaddr
```
## Example

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
