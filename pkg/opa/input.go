package opa

import "net/http"

func InputFromHTTPRequest(r *http.Request) map[string]interface{} {
	return map[string]interface{}{
		"method":         r.Method,
		"host":           r.Host,
		"proto":          r.Proto,
		"url":            r.URL.String(),
		"request_uri":    r.RequestURI,
		"remote_addr":    r.RemoteAddr,
		"content_length": r.ContentLength,
		"headers":        r.Header,
	}
}
