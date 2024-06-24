package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

type UpstreamClientOptions struct {
	DNSServers []DNSServer
	Host       string
	Port       string
}

type DNSServer struct {
	Network string
	Addr    string
}

func NewUpsteamClient(opts UpstreamClientOptions) *http.Client {
	upstreamServerHost := opts.Host
	if opts.Host == "" {
		upstreamServerHost = "localhost"
	}

	upstreamServerPort := opts.Port
	if opts.Port == "" {
		upstreamServerPort = "80"
	}

	customResolver := &net.Resolver{
		PreferGo: true,
		Dial: func(_ context.Context, _, _ string) (net.Conn, error) {
			var conn net.Conn
			for _, dnsServer := range opts.DNSServers {
				dnsNetwork := dnsServer.Network
				if dnsNetwork == "" {
					dnsNetwork = "tcp6"
				}

				dnsAddr := dnsServer.Addr
				if dnsAddr == "" {
					dnsAddr = "[::1]:53"
				}

				var err error
				conn, err = net.DialTimeout(dnsNetwork, dnsAddr, time.Second)
				if err != nil {
					return nil, fmt.Errorf("failed to dial dns server: %w", err)
				}

				if conn != nil {
					break
				}
			}

			if conn == nil {
				return nil, errors.New("failed to dial all dns servers")
			}

			return conn, nil
		},
	}

	dialer := net.Dialer{}

	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network string, _ string) (net.Conn, error) {
				ips, err := customResolver.LookupIP(ctx, "ip", upstreamServerHost)
				if err != nil {
					return nil, fmt.Errorf("failed to lookup IP: %w", err)
				}

				var conn net.Conn
				for _, ip := range ips {
					upstreamServerAddrAndPort := net.JoinHostPort(ip.String(), upstreamServerPort)

					conn, err = dialer.DialContext(ctx, network, upstreamServerAddrAndPort)
					if err != nil {
						return nil, fmt.Errorf("failed to dial upstream server: %w", err)
					}

					if conn != nil {
						break
					}
				}

				return conn, nil
			},
		},
	}
}
