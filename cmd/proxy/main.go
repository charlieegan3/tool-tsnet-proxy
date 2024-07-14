package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/miekg/dns"

	"github.com/charlieegan3/tool-tsnet-proxy/pkg/proxy"
)

func main() {
	configFilePath := "config.yaml"
	if len(os.Args) > 1 {
		configFilePath = os.Args[1]
	}

	configFile, err := os.Open(configFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open config file: %v\n", err)

		return
	}

	cfg, err := proxy.LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)

		return
	}

	cfgCtx, cfgCtxCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cfgCtxCancel()

	proxyHandler, dnsServers, err := proxy.NewHandlerFromConfig(cfgCtx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create proxy: %v\n", err)

		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, dnsServer := range dnsServers {
		go func(dnsServer *dns.Server) {
			select {
			case <-ctx.Done():
				shutdownContext, shutdownContextCancel := context.WithTimeout(ctx, 2*time.Second)
				defer shutdownContextCancel()

				if err := dnsServer.ShutdownContext(shutdownContext); err != nil {
					log.Fatalf("Failed to shutdown DNS server: %s\n", err.Error())
				}
			default:
				if err := dnsServer.ListenAndServe(); err != nil {
					log.Fatalf("Failed to start DNS server: %s\n", err.Error())
				}
			}
		}(dnsServer)
	}

	srv := &http.Server{
		Addr:    net.JoinHostPort(cfg.Addr, strconv.Itoa(cfg.Port)),
		Handler: proxyHandler,

		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		select {
		case <-ctx.Done():
			shutdownContext, shutdownContextCancel := context.WithTimeout(ctx, 2*time.Second)
			defer shutdownContextCancel()

			if err := srv.Shutdown(shutdownContext); err != nil {
				log.Fatalf("Failed to shutdown proxy server: %s\n", err.Error())
			}
		default:
			if err := srv.ListenAndServe(); err != nil {
				log.Fatalf("Failed to start proxy server: %s\n", err.Error())
			}
		}
	}()

	fmt.Fprintln(os.Stderr, "Proxy started")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		return
	case <-sigChan:
		fmt.Fprintln(os.Stderr, "\nShutting down...")
		cancel()
	}
}
