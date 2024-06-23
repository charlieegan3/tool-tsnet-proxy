package doh

import (
	"context"
	"fmt"
	"time"

	"github.com/miekg/dns"
)

type WrappingDNSServerOptions struct {
	Addr       string
	DoHServers []string
	Timeout    time.Duration
}

func NewWrappingDNSServer(opts *WrappingDNSServerOptions) *dns.Server {
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

		ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
		defer cancel()

		select {
		case <-ctx.Done():
			dns.HandleFailed(w, r)
		default:
			for _, dohServer := range opts.DoHServers {
				records, err := QueryA(ctx, dohServer, n)
				if err != nil {
					continue
				}

				for _, record := range records {
					rr, _ := dns.NewRR(fmt.Sprintf("%s 3600 IN A %s", n, record))
					m.Answer = append(m.Answer, rr)
				}

				break
			}
		}
	})

	return &dns.Server{
		Addr:    opts.Addr,
		Net:     "tcp",
		Handler: dnsMux,
	}
}
