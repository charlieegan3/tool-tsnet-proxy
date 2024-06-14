package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/open-policy-agent/opa/sdk"

	"github.com/charlieegan3/tool-tsnet-proxy/pkg/httpclient"
	opa "github.com/charlieegan3/tool-tsnet-proxy/pkg/opa"
	dnstest "github.com/charlieegan3/tool-tsnet-proxy/pkg/test/dns"
	opatest "github.com/charlieegan3/tool-tsnet-proxy/pkg/test/opa"

	_ "embed"
)

//go:embed fixtures/policy.rego
var regoPolicy string

func TestProxyWithTwoMatchers(t *testing.T) {
	t.Parallel()

	const (
		upstreamServerHost1 = "1.example.com"
		upstreamServerHost2 = "2.example.com"
		externalHost1       = "ext1.example.com"
		externalHost2       = "ext2.example.com"
	)

	// this test dns server will map example.com to the loopback address
	// where the test servers are running
	dnsServer, err := dnstest.NewServer(dnstest.Options{
		MappingsA: map[string]string{
			upstreamServerHost1 + ".": "127.0.0.1",
			upstreamServerHost2 + ".": "127.0.0.1",
			externalHost1 + ".":       "127.0.0.1",
			externalHost2 + ".":       "127.0.0.1",
		},
	})
	if err != nil {
		t.Fatalf("Failed to start DNS server: %s", err)
	}
	//nolint:errcheck
	defer dnsServer.Shutdown()

	// upstream servers that are running behind the proxy
	upstreamServerHandler1 := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("1"))
		if err != nil {
			t.Fatalf("Failed to write response: %s", err)
		}
	})
	upstreamServer1, _, upstreamServerClient1 := newTestServerAndClient(
		upstreamServerHost1,
		upstreamServerHandler1,
		dnsServer,
	)

	defer upstreamServer1.Close()

	// function that will match requests to the upstream servers
	upstreamMatcher1 := func(req *http.Request) (*http.Client, bool) {
		if req.URL.Path != "/foobar" {
			return nil, false
		}

		if !strings.HasPrefix(req.Host, externalHost1) {
			return nil, false
		}

		return upstreamServerClient1, true
	}

	// upstream servers that are running behind the proxy
	upstreamServerHandler2 := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("2"))
		if err != nil {
			t.Fatalf("Failed to write response: %s", err)
		}
	})
	upstreamServer2, _, upstreamServerClient2 := newTestServerAndClient(
		upstreamServerHost2,
		upstreamServerHandler2,
		dnsServer,
	)

	defer upstreamServer2.Close()

	// function that will match requests to the upstream servers
	upstreamMatcher2 := func(req *http.Request) (*http.Client, bool) {
		if req.URL.Path != "/foobar" {
			return nil, false
		}

		if !strings.HasPrefix(req.Host, externalHost2) {
			return nil, false
		}

		return upstreamServerClient2, true
	}

	// create the proxy handler and server
	opts := &Options{
		Matchers: []Matcher{upstreamMatcher2, upstreamMatcher1},
	}

	proxyHandler, err := NewHandler(opts)
	if err != nil {
		t.Fatalf("Failed to create proxy handler: %v", err)
	}

	proxyServer, proxyServerURL, proxyServerClient := newTestServerAndClient(
		externalHost1,
		proxyHandler,
		dnsServer,
	)
	defer proxyServer.Close()

	// make an example request to the proxy server to upstreamServer1
	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second)
	defer cancel1()

	req, err := http.NewRequestWithContext(ctx1, http.MethodGet, fmt.Sprintf(
		"http://%s/foobar",
		net.JoinHostPort(externalHost1, proxyServerURL.Port()),
	), nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Host", upstreamServerHost1)

	resp1, err := proxyServerClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp1.Body.Close()

	assertStatusAndContent(t, resp1, http.StatusOK, "1")

	// make an example request to the proxy server to upstreamServer2
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()

	req, err = http.NewRequestWithContext(ctx2, http.MethodGet, fmt.Sprintf(
		"http://%s/foobar",
		net.JoinHostPort(externalHost2, proxyServerURL.Port()),
	), nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Host", upstreamServerHost2)

	resp2, err := proxyServerClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	assertStatusAndContent(t, resp2, http.StatusOK, "2")
}

func TestProxyWithMiddlewares(t *testing.T) {
	t.Parallel()

	const (
		upstreamServerHost = "example.com"
		externalHost       = "proxy.example.com"
	)

	dnsServer, err := dnstest.NewServer(dnstest.Options{
		MappingsA: map[string]string{
			upstreamServerHost + ".": "127.0.0.1",
			externalHost + ".":       "127.0.0.1",
		},
	})
	if err != nil {
		t.Fatalf("Failed to start DNS server: %s", err)
	}
	//nolint:errcheck
	defer dnsServer.Shutdown()

	// upstream servers that are running behind the proxy
	upstreamServerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testHeader := r.Header.Get("X-Test")

		_, err := w.Write([]byte(testHeader))
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

		if !strings.HasPrefix(req.Host, externalHost) {
			return nil, false
		}

		return upstreamServerClient, true
	}

	addHeaderMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Test", "test")

			next.ServeHTTP(w, r)
		})
	}

	bundleServer, err := opatest.NewBundleServer(map[string][]byte{
		"policy1.rego": []byte(regoPolicy),
	})
	if err != nil {
		t.Fatalf("Failed to create bundle server: %v", err)
	}

	defer bundleServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	opaInstance, err := opa.NewInstance(ctx, opa.InstanceOptions{
		BundleServerAddr: bundleServer.URL,
		BundlePath:       "bundle.tar.gz",
	})
	if err != nil {
		t.Fatalf("Failed to create OPA instance: %v", err)
	}

	policyMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rs, err := opaInstance.Decision(ctx, sdk.DecisionOptions{
				Path:  "authz/allow",
				Input: opa.InputFromHTTPRequest(r),
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)

				return
			}

			if rs == nil || rs.Result == nil {
				http.Error(w, "not allowed", http.StatusForbidden)

				return
			}

			allowed, ok := rs.Result.(bool)
			if !ok {
				http.Error(w, "not allowed", http.StatusForbidden)

				return
			}

			if !allowed {
				http.Error(w, "not allowed", http.StatusForbidden)

				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// create the proxy handler and server
	opts := &Options{
		Matchers:    []Matcher{upstreamMatcher},
		Middlewares: []Middleware{addHeaderMiddleware, policyMiddleware},
	}

	proxyHandler, err := NewHandler(opts)
	if err != nil {
		t.Fatalf("Failed to create proxy handler: %v", err)
	}

	proxyServer, proxyServerURL, proxyServerClient := newTestServerAndClient(
		externalHost,
		proxyHandler,
		dnsServer,
	)
	defer proxyServer.Close()

	// make an example request to the proxy server to upstreamServer1
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()

	req, err := http.NewRequestWithContext(ctx2, http.MethodGet, fmt.Sprintf(
		"http://%s/foobar",
		net.JoinHostPort(externalHost, proxyServerURL.Port()),
	), nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Host", upstreamServerHost)

	resp, err := proxyServerClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	assertStatusAndContent(t, resp, http.StatusOK, "test")
}

func assertStatusAndContent(t *testing.T, resp *http.Response, status int, content string) {
	t.Helper()

	bodyBs, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != status {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}

	if !strings.Contains(string(bodyBs), content) {
		t.Errorf("Expected 'Hello, client', got %s", string(bodyBs))
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
