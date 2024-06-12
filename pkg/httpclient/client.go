package httpclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

type UpstreamClientOptions struct {
	DNSNetwork string
	DNSAddr    string
	Host       string
	Port       string
}

func NewUpsteamClient(opts UpstreamClientOptions) *http.Client {
	dnsNetwork := opts.DNSNetwork
	if dnsNetwork == "" {
		dnsNetwork = "tcp6"
	}

	dnsAddr := opts.DNSAddr
	if dnsAddr == "" {
		dnsAddr = "[::1]:53"
	}

	upstreamServerHost := opts.Host
	if opts.Host == "" {
		upstreamServerHost = "localhost"
	}

	upstreamServerPort := opts.Port
	if opts.Port == "" {
		upstreamServerPort = "80"
	}

	upstreamServerAddrAndPort := fmt.Sprintf("%s:%s", upstreamServerHost, upstreamServerPort)

	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network string, _ string) (net.Conn, error) {
				customDialer := &net.Dialer{
					Resolver: &net.Resolver{
						PreferGo: true,
						Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
							conn, err := net.Dial(dnsNetwork, dnsAddr)
							if err != nil {
								return nil, fmt.Errorf("failed to dial dns server: %w", err)
							}

							return conn, nil
						},
					},
				}

				conn, err := customDialer.DialContext(ctx, network, upstreamServerAddrAndPort)
				if err != nil {
					return nil, fmt.Errorf("failed to dial upstream server: %w", err)
				}

				return conn, nil
			},
		},
	}
}
