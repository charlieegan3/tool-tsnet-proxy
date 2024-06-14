package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	opatest "github.com/charlieegan3/tool-tsnet-proxy/pkg/test/opa"
)

func TestMiddlewareFromConfigMiddleware(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create a test OPA server
	module := `
		package authz

		default allow = false

		allow {
			input.method == "GET"
		}
	`

	bundleServer, err := opatest.NewBundleServer(map[string][]byte{
		"policy.rego": []byte(module),
	})
	if err != nil {
		t.Fatalf("Failed to create bundle server: %v", err)
	}
	defer bundleServer.Close()

	// Define a sample ConfigMiddleware
	config := ConfigMiddleware{
		Kind: "opa",
		OPAProperties: &ConfigMiddlewarePropsOPA{
			Bundle: ConfigMiddlewarePropsOPABundle{
				ServerEndpoint: bundleServer.URL,
				Path:           "bundle.tar.gz",
			},
		},
	}

	// Convert the config to a middleware
	middleware, err := MiddlewareFromConfigMiddleware(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create middleware from config: %v", err)
	}

	// Create a test handler to wrap with the middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the handler with the middleware
	wrappedHandler := middleware(handler)

	// Create a test server
	server := httptest.NewServer(wrappedHandler)
	defer server.Close()

	// Define test cases
	tests := []struct {
		method   string
		expected int
	}{
		{"GET", http.StatusOK},
		{"POST", http.StatusForbidden},
	}

	for _, test := range tests {
		req, err := http.NewRequestWithContext(ctx, test.method, server.URL, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != test.expected {
			t.Errorf("Expected status %d, got %d", test.expected, resp.StatusCode)
		}
	}
}
