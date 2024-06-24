package doh

import (
	"context"
	"fmt"
	"log"
	"net"
	"slices"
	"strconv"
	"testing"
	"time"

	dohtest "github.com/charlieegan3/tool-tsnet-proxy/pkg/test/doh"
	"github.com/charlieegan3/tool-tsnet-proxy/pkg/utils"
)

func TestNewWrappingDNSServer(t *testing.T) {
	t.Parallel()

	dohServer := dohtest.NewLocalDOHServer(
		map[string]string{
			"example.com": "127.0.0.1",
		},
	)
	defer dohServer.Close()

	freePort, err := utils.FreePort(0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dnsServer := NewWrappingDNSServer(
		&WrappingDNSServerOptions{
			Addr:       net.JoinHostPort("localhost", strconv.Itoa(freePort)),
			DoHServers: []string{dohServer.URL},
			Timeout:    1 * time.Second,
		},
	)
	go func() {
		if err := dnsServer.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start DNS server: %s\n", err.Error())
		}
	}()
	//nolint:errcheck
	defer dnsServer.Shutdown()

	customResolver := &net.Resolver{
		PreferGo: true,
		Dial: func(_ context.Context, _, _ string) (net.Conn, error) {
			conn, err := net.Dial("tcp", dnsServer.Addr)
			if err != nil {
				return nil, fmt.Errorf("failed to dial dns server: %w", err)
			}

			return conn, nil
		},
	}

	netIPs, err := customResolver.LookupIP(context.Background(), "ip", "example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	ips := make([]string, 0, len(netIPs))

	for _, ip := range netIPs {
		ips = append(ips, ip.String())
	}

	if exp, got := []string{"127.0.0.1"}, ips; !slices.Equal(exp, got) {
		t.Fatalf("expected %v, got %v", exp, got)
	}
}
