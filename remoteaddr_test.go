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
