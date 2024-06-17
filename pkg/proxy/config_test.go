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
	}
}
