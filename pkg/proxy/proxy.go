package proxy

import (
	"fmt"
	"io"
	"net/http"
)

type Options struct {
	Matchers []Matcher
}

type Matcher func(*http.Request) (*http.Client, bool)

func NewHandler(opts *Options) http.Handler {
	return &proxy{
		matchers: opts.Matchers,
	}
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

	// RequestURI nust be cleared to be accedpted by the client.Do function.
	r.RequestURI = ""
	// here the host and port will be determined by the client, however,
	// the host and scheme must be set to pass validation.
	r.URL, _ = r.URL.Parse("http://surplus")

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

	return
}
