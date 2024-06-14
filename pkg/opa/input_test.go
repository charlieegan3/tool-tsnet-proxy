package opa

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestInputFromHTTPRequest(t *testing.T) {
	t.Parallel()

	req := &http.Request{
		Method: http.MethodGet,
		Proto:  "HTTP/1.1",
		URL: &url.URL{
			Path: "/example_path",
		},
		Host: "example.com",
		Header: http.Header{
			"Host":     {"example.com"},
			"X-FooBar": {"wow"},
		},
		RequestURI:    "/example_path",
		RemoteAddr:    "127.0.0.1:1234",
		ContentLength: 100000,
	}

	input := InputFromHTTPRequest(req)
	exp := map[string]any{
		"method":         http.MethodGet,
		"host":           "example.com",
		"proto":          "HTTP/1.1",
		"url":            "/example_path",
		"request_uri":    "/example_path",
		"remote_addr":    "127.0.0.1:1234",
		"content_length": int64(100000),
		"headers": http.Header{
			"Host":     {"example.com"},
			"X-FooBar": {"wow"},
		},
	}

	if !reflect.DeepEqual(input, exp) {
		for k := range exp {
			if !reflect.DeepEqual(input[k], exp[k]) {
				if reflect.TypeOf(input[k]) != reflect.TypeOf(exp[k]) {
					t.Errorf("Key %q: type mismatch - want type %T, got type %T", k, exp[k], input[k])
				} else {
					t.Errorf("Key %q: want %v, got %v", k, exp[k], input[k])
				}
			}
		}

		for k := range input {
			if _, found := exp[k]; !found {
				t.Errorf("Unexpected key %q in result with value %v", k, input[k])
			}
		}
	}
}
