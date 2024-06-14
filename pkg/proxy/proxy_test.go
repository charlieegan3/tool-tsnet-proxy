package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/charlieegan3/tool-tsnet-proxy/pkg/httpclient"
	dnstest "github.com/charlieegan3/tool-tsnet-proxy/pkg/test/dns"
	"github.com/miekg/dns"
)

func TestSimple(t *testing.T) {
	// example.com is the example upstream server host
	const upstreamServerHost = "example.com"

	const proxyServerExternalHost = "proxy.example.com"

	// this test dns server will map example.com to the loopback address
	// where the test servers are running
	dnsServer, err := dnstest.NewServer(dnstest.Options{
		MappingsA: map[string]string{
			upstreamServerHost + ".":      "127.0.0.1",
			proxyServerExternalHost + ".": "127.0.0.1",
		},
	})
	if err != nil {
		t.Fatalf("Failed to start DNS server: %s", err)
	}
	defer dnsServer.Shutdown()

	// upstream server that is running behind the proxy
	upstreamServerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Hello, client"))
		if err != nil {
			t.Fatalf("Failed to write response: %s", err)
		}
	})
	upstreamServer, _, upstreamServerClient := newTestServerAndClient(
		upstreamServerHost,
		upstreamServerHandler,
		dnsServer,
	)
	defer upstreamServer.Close()

	// function that will match requests to the upstream servers
	upstreamMatcher := func(req *http.Request) (*http.Client, bool) {
		if req.URL.Path != "/foobar" {
			return nil, false
		}

		if !strings.HasPrefix(req.Host, proxyServerExternalHost) {
			return nil, false
		}

		return upstreamServerClient, true
	}

	// create the proxy handler and server
	opts := &Options{
		Matchers: []Matcher{upstreamMatcher},
	}

	proxyServer, proxyServerURL, proxyServerClient := newTestServerAndClient(
		proxyServerExternalHost,
		NewHandler(opts),
		dnsServer,
	)
	defer proxyServer.Close()

	// make an example request to the proxy server
	// the request will be matched based on the path. Host routing
	// is not used as it didn't seem to work with httptest
	req, err := http.NewRequest("GET", fmt.Sprintf(
		"http://%s:%s/foobar",
		proxyServerExternalHost,
		proxyServerURL.Port(),
	), nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Host", upstreamServerHost)

	resp, err := proxyServerClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	assertStatusAndContent(t, resp, 200, "Hello, client")
}

func assertStatusAndContent(t *testing.T, resp *http.Response, status int, content string) {
	bodyBs, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != status {
		t.Log(string(bodyBs))

		t.Fatalf("Expected status code 200, got %d", resp.StatusCode)
	}

	if !strings.Contains(string(bodyBs), content) {
		t.Fatalf("Expected 'Hello, client', got %s", string(bodyBs))
	}
}

func newTestServerAndClient(
	host string,
	handler http.Handler,
	dnsServer *dns.Server,
) (*httptest.Server, *url.URL, *http.Client) {
	s := httptest.NewServer(handler)

	sURL, _ := url.Parse(s.URL)

	// client that is configured to send all requests to the upstream server
	c := httpclient.NewUpsteamClient(httpclient.UpstreamClientOptions{
		DNSNetwork: dnsServer.Net,
		DNSAddr:    dnsServer.Addr,
		Host:       host,
		Port:       sURL.Port(),
	})

	return s, sURL, c
}
