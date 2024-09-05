package proxy

import (
	"slices"
	"strings"
	"testing"

	_ "embed"
)

//go:embed fixtures/config.yaml
var configYAML string

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	reader := strings.NewReader(configYAML)

	cfg, err := LoadConfig(reader)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if exp, got := "localhost", cfg.Addr; exp != got {
		t.Fatalf("ListenAddr did not match expected: %q != %q", exp, got)
	}

	if exp, got := 8080, cfg.Port; exp != got {
		t.Fatalf("Port did not match expected: %d != %d", exp, got)
	}

	if exp, got := "proxy.example.com", cfg.Host; exp != got {
		t.Fatalf("Host did not match expected: %q != %q", exp, got)
	}

	expectedDNSServers := []ConfigDNSServer{
		{Addr: "::1", Net: "udp6"},
		{Addr: "[::1]:53", Net: "tcp6"},
		{Addr: "1.1.1.1", Net: "udp4"},
		{Addr: "1.1.1.1:53", Net: "tcp4"},
	}

	if !slices.Equal(cfg.DNSServers, expectedDNSServers) {
		t.Fatalf("DNSServers did not match expected")
	}

	expectedMiddlewares := []ConfigMiddleware{
		{
			Kind: "opa",
			OPAProperties: &ConfigMiddlewarePropsOPA{
				Bundle: ConfigMiddlewarePropsOPABundle{
					ServerEndpoint: "https://example.com",
					Path:           "/bundles/policy.tar.gz",
				},
			},
		},
	}

	if len(cfg.Middlewares) != len(expectedMiddlewares) {
		t.Fatalf("Middlewares length did not match expected")
	}

	for i, mw := range cfg.Middlewares {
		if mw.Kind != expectedMiddlewares[i].Kind {
			t.Fatalf("Middleware kind did not match expected")
		}

		if mw.OPAProperties.Bundle.ServerEndpoint != expectedMiddlewares[i].OPAProperties.Bundle.ServerEndpoint {
			t.Fatalf("Middleware bundle server-endpoint did not match expected")
		}

		if mw.OPAProperties.Bundle.Path != expectedMiddlewares[i].OPAProperties.Bundle.Path {
			t.Fatalf("Middleware bundle path did not match expected")
		}
	}

	expectedUpstreams := []ConfigUpstream{
		{
			Endpoint: "http://internal.example.com",
			Hosts: []string{
				"foo.example.com",
				"bar.example.com",
			},
			PathPrefixes: []string{
				"/foo",
				"/bar",
			},
			Tailnet: "tsnet",
		},
		{
			Endpoint: "http://internal2.example.com",
			Hosts: []string{
				"foo2.example.com",
				"bar2.example.com",
			},
			PathPrefixes: []string{
				"/foo2",
				"/bar2",
			},
		},
	}

	if len(cfg.Upstreams) != len(expectedUpstreams) {
		t.Fatalf("Upstreams length did not match expected")
	}

	for i, us := range cfg.Upstreams {
		if us.Endpoint != expectedUpstreams[i].Endpoint {
			t.Fatalf("Upstream endpoint did not match expected")
		}

		if !slices.Equal(us.Hosts, expectedUpstreams[i].Hosts) {
			t.Fatalf("Upstream hosts did not match expected")
		}

		if !slices.Equal(us.PathPrefixes, expectedUpstreams[i].PathPrefixes) {
			t.Fatalf("Upstream path-prefixes did not match expected")
		}

		if us.Tailnet != expectedUpstreams[i].Tailnet {
			t.Fatalf("Upstream tailnet did not match expected")
		}
	}

	expectedTailnets := map[string]ConfigTailnet{
		"foobar": {
			ID:      "proxy-box",
			AuthKey: "example",
		},
	}

	for k, v := range cfg.Tailnets {
		tn, ok := expectedTailnets[k]
		if !ok {
			t.Fatalf("Key %s not found in expected tailnets", k)
		}

		if v.AuthKey != tn.AuthKey {
			t.Fatalf("Auth key value %q for tailnet %q did not match expected %q", v.AuthKey, k, tn.AuthKey)
		}

		if v.ID != tn.ID {
			t.Fatalf("ID value %q for tailnet %q did not match expected %q", v.ID, k, tn.ID)
		}
	}

	if exp, got := "https://example.com/callback", cfg.OAuth.CallbackURL; exp != got {
		t.Fatalf("OAuth callback url did not match expected: %q != %q", exp, got)
	}

	if exp, got := "https://foo.example.com", cfg.OAuth.ProviderURL; exp != got {
		t.Fatalf("OAuth provider URL did not match expected: %q != %q", exp, got)
	}

	if exp, got := "proxy-foo", cfg.OAuth.ClientID; exp != got {
		t.Fatalf("OAuth client ID did not match expected: %q != %q", exp, got)
	}

	if exp, got := "secretsecretsecret", cfg.OAuth.ClientSecret; exp != got {
		t.Fatalf("OAuth client secret did not match expected: %q != %q", exp, got)
	}
}
