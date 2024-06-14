package proxy

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

type Options struct {
	Matchers    []Matcher
	Middlewares []Middleware
}

type Matcher func(*http.Request) (*http.Client, bool)

type Middleware func(http.Handler) http.Handler

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

	// RequestURI must be cleared to be accepted by the client.Do function.
	r.RequestURI = ""
	// here the host and port will be determined by the client, however,
	// the host and scheme must be set to pass validation.
	r.URL, _ = r.URL.Parse("http://host-is-ignored")

	resp, err := client.Do(r)
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
