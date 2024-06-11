package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	doh_test "github.com/charlieegan3/tool-tsnet-proxy/pkg/doh/test"
)

func TestSimpleUseCase(t *testing.T) {

	dohServer := doh_test.NewLocalDOHServer(
		map[string]string{
			"example.com": "127.0.0.1",
		},
	)
	defer dohServer.Close()

	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, client"))
	}))

	backendServerClient := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network string, _ string) (net.Conn, error) {
				customDialer := &net.Dialer{
					Resolver: &net.Resolver{
						PreferGo: true,
						Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
							conn, err := net.Dial(network, dohServer.URL)
							if err != nil {
								return nil, fmt.Errorf("failed to dial DoH server: %w", err)
							}

							return conn, nil
						},
					},
				}

				conn, err := customDialer.DialContext(ctx, network, strings.TrimPrefix(backendServer.URL, "http://"))
				if err != nil {
					return nil, fmt.Errorf("failed to dial backend server: %w", err)
				}

				return conn, nil
			},
		},
	}

	backendServerMatcher := func(req *http.Request) (*http.Client, bool) {
		if req.URL.Path == "/foobar" {
			return &backendServerClient, true
		}

		return nil, false
	}

	opts := &Options{
		Matchers: []Matcher{
			backendServerMatcher,
		},
	}

	h := NewHandler(opts)

	proxyServer := httptest.NewServer(h)
	defer proxyServer.Close()

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/foobar", proxyServer.URL), nil)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
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
