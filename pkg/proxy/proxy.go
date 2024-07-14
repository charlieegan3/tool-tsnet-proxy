package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/miekg/dns"
	"tailscale.com/tsnet"

	"github.com/charlieegan3/tool-tsnet-proxy/pkg/doh"
	"github.com/charlieegan3/tool-tsnet-proxy/pkg/httpclient"
	"github.com/charlieegan3/tool-tsnet-proxy/pkg/utils"
)

func NewHandlerFromConfig(ctx context.Context, config *Config) (
	http.Handler,
	[]*dns.Server,
	error,
) {
	dnsServers := make([]httpclient.DNSServer, 0)
	wrappedDNSServers := make([]*dns.Server, 0)

	for _, dnsServer := range config.DNSServers {
		if dnsServer.DoH {
			wrappedDNSServerPort, err := utils.FreePort(0)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to find free port for wrapped DNS server: %w", err)
			}

			addr := net.JoinHostPort("localhost", strconv.Itoa(wrappedDNSServerPort))

			//nolint:contextcheck
			dnsServer := doh.NewWrappingDNSServer(
				&doh.WrappingDNSServerOptions{
					Addr:       addr,
					DoHServers: []string{dnsServer.Addr},
					Timeout:    1 * time.Second,
				},
			)

			wrappedDNSServers = append(wrappedDNSServers, dnsServer)

			dnsServers = append(dnsServers, httpclient.DNSServer{
				Addr:    addr,
				Network: "tcp",
			})

			continue
		}

		dnsServers = append(dnsServers, httpclient.DNSServer{
			Network: dnsServer.Net,
			Addr:    dnsServer.Addr,
		})
	}

	tsNetServers := make(map[string]*tsnet.Server)
	for k, tnet := range config.Tailnets {
		tsNetServers[k] = &tsnet.Server{
			Hostname: tnet.ID,
			AuthKey:  tnet.AuthKey,
		}
	}

	// Create matchers from upstreams
	matchers := make([]Matcher, 0)

	for _, upstream := range config.Upstreams {
		upstreamURL, err := url.Parse(upstream.Endpoint)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse upstream URL: %w", err)
		}

		if upstreamURL.Scheme == "https" && upstreamURL.Port() == "" {
			upstreamURL.Host = upstreamURL.Host + ":443"
		}

		var dialFunc func(context.Context, string, string) (net.Conn, error)

		if upstream.Tailnet != "" {
			tsNetServer, ok := tsNetServers[upstream.Tailnet]
			if !ok {
				return nil, nil, fmt.Errorf("tailnet %s not found", upstream.Tailnet)
			}

			dialFunc = tsNetServer.Dial
		}

		client := httpclient.NewUpsteamClient(httpclient.UpstreamClientOptions{
			Host:       upstreamURL.Hostname(),
			Port:       upstreamURL.Port(),
			DNSServers: dnsServers,
			DialFunc:   dialFunc,
		})
		matcher := MatcherFromUpstream(upstream, client)
		matchers = append(matchers, matcher)
	}

	// Create middlewares from config middlewares
	middlewares := make([]Middleware, 0)

	for _, configMiddleware := range config.Middlewares {
		middleware, err := MiddlewareFromConfigMiddleware(ctx, configMiddleware)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create middleware: %w", err)
		}

		middlewares = append(middlewares, middleware)
	}

	// Create a new proxy handler with the matchers and middlewares
	handler, err := NewHandler(&Options{
		Matchers:    matchers,
		Middlewares: middlewares,
	})
	if err != nil {
		return nil, nil, err
	}

	return handler, wrappedDNSServers, nil
}

func NewHandler(opts *Options) (http.Handler, error) {
	var handler http.Handler
	handler = &proxy{
		matchers: opts.Matchers,
	}

	// middlewares are applied in reverse order to they are called
	// in the order that they appear in the slice.
	for i := len(opts.Middlewares) - 1; i >= 0; i-- {
		handler = opts.Middlewares[i](handler)
		if handler == nil {
			return nil, errors.New("middleware returned nil handler")
		}
	}

	return handler, nil
}

type proxy struct {
	matchers []Matcher
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var client *http.Client

	for _, matcher := range p.matchers {
		var ok bool

		client, ok = matcher(r)
		if ok {
			break
		}
	}

	if client == nil {
		http.Error(w, "not found", http.StatusNotFound)

		return
	}

	h := r.Header

	rURL, err := url.Parse(fmt.Sprintf("https://charlieegan3.com%s", r.URL.Path))

	req := &http.Request{
		Method: r.Method,
		Header: h,
		URL:    rURL,
		Body:   r.Body,
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w,
			fmt.Errorf("failed to send request: %w", err).Error(),
			http.StatusBadGateway,
		)

		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}

	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(
			w,
			fmt.Errorf("failed to copy response body: %w", err).Error(),
			http.StatusBadGateway,
		)

		return
	}
}
