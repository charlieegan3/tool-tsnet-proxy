package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
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

	upstreamServerAddrAndPort := fmt.Sprintf("%s:%s", upstreamServerHost, upstreamServerPort)

	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network string, _ string) (net.Conn, error) {
				customDialer := &net.Dialer{
					Resolver: &net.Resolver{
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
								conn, err = net.Dial(dnsNetwork, dnsAddr)
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
