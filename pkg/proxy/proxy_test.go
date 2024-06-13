package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/charlieegan3/tool-tsnet-proxy/pkg/httpclient"
	dnstest "github.com/charlieegan3/tool-tsnet-proxy/pkg/test/dns"
)

func TestSimple(t *testing.T) {
	// example.com is the example upstream server host
	const upstreamServerHost = "example.com"

	// upstream server that is running behind the proxy
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range r.Header {
			fmt.Println(k, v)
		}
		_, err := w.Write([]byte("Hello, client"))
		if err != nil {
			t.Fatalf("Failed to write response: %s", err)
		}
	}))

	upstreamServerURL, err := url.Parse(upstreamServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse backend server URL: %s", err)
	}

	// this test dns server will map example.com to the loopback address
	dnsServer, err := dnstest.NewServer(dnstest.Options{
		MappingsA: map[string]string{
			upstreamServerHost + ".": "127.0.0.1",
		},
	})
	if err != nil {
		t.Fatalf("Failed to start DNS server: %s", err)
	}
	defer dnsServer.Shutdown()

	// client that is configured to send all requests to the upstream server
	upstreamServerClient := httpclient.NewUpsteamClient(httpclient.UpstreamClientOptions{
		DNSNetwork: dnsServer.Net,
		DNSAddr:    dnsServer.Addr,
		Host:       upstreamServerHost,
		Port:       upstreamServerURL.Port(),
	})

	// function that will match requests to the upstream servers
	upstreamMatcher := func(req *http.Request) (*http.Client, bool) {
		if req.URL.Path == "/foobar" {
			return upstreamServerClient, true
		}

		return nil, false
	}

	// create the proxy handler
	opts := &Options{
		Matchers: []Matcher{upstreamMatcher},
	}

	h := NewHandler(opts)

	// start the proxy server within an http test server
	proxyServer := httptest.NewServer(h)
	defer proxyServer.Close()

	proxyServerURL, err := url.Parse(proxyServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse proxy server URL: %s", err)
	}

	// client that is configured to send all requests to the proxy, regardless of the host
	proxyServerClient := httpclient.NewUpsteamClient(httpclient.UpstreamClientOptions{
		DNSNetwork: dnsServer.Net,
		DNSAddr:    dnsServer.Addr,
		Host:       upstreamServerHost,
		Port:       proxyServerURL.Port(),
	})

	// make an example request to the proxy server
	// the request will be matched based on the path. Host routing
	// is not used as it didn't seem to work with httptest
	req, err := http.NewRequest("GET", fmt.Sprintf("http://example.com:%s/foobar", proxyServerURL.Port()), nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Host", upstreamServerHost)

	resp, err := proxyServerClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	bodyBs, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Log(string(bodyBs))

		t.Fatalf("Expected status code 200, got %d", resp.StatusCode)
	}

	if string(bodyBs) != "Hello, client" {
		t.Fatalf("Expected 'Hello, client', got %s", string(bodyBs))
	}
}
