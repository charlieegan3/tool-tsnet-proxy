package dnstest

import (
	"fmt"
	"log"

	utilstest "github.com/charlieegan3/tool-tsnet-proxy/pkg/test/utils"
	"github.com/miekg/dns"
)

type Options struct {
	Network   string
	Addr      string
	Port      int
	MappingsA map[string]string
}

func NewServer(opts Options) (*dns.Server, error) {
	if opts.Port == 0 {
		var err error

		opts.Port, err = utilstest.FreePort(0)
		if err != nil {
			return nil, fmt.Errorf("failed to get free port: %w", err)
		}
	}

	if opts.Network == "" {
		opts.Network = "tcp6"
	}

	if opts.Addr == "" {
		opts.Addr = "::1"
	}

	dnsMux := dns.NewServeMux()

	dnsMux.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		//nolint:errcheck
		defer w.WriteMsg(m)

		m.SetReply(r)
		m.Authoritative = true

		if len(r.Question) == 0 {
			return
		}

		if r.Question[0].Qtype != dns.TypeA {
			return
		}

		n := r.Question[0].Name

		if _, ok := opts.MappingsA[n]; ok {
			rr, _ := dns.NewRR(fmt.Sprintf("%s 3600 IN A %s", n, opts.MappingsA[n]))
			m.Answer = append(m.Answer, rr)
		}

		//nolint:errcheck
		w.WriteMsg(m)
	})

	server := &dns.Server{
		Addr:    fmt.Sprintf("[%s]:%d", opts.Addr, opts.Port),
		Net:     "tcp6",
		Handler: dnsMux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start DNS server: %s\n", err.Error())
		}
	}()

	return server, nil
}
