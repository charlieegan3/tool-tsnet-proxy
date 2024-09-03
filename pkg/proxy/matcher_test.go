package proxy

import (
	"net/http"
	"net/url"
	"testing"
)

func TestMatcherFromUpstream(t *testing.T) {
	t.Parallel()

	client := &http.Client{}

	upstream := ConfigUpstream{
		Endpoint: "http://internal.example.com",
		Hosts: []string{
			"foo.example.com",
			"bar.example.com",
		},
		PathPrefixes: []string{
			"/foo",
			"/bar",
		},
	}

	matcher := MatcherFromUpstream(upstream, client)

	tests := []struct {
		req    *http.Request
		expect bool
	}{
		{
			req: &http.Request{
				Host: "foo.example.com",
				URL:  &url.URL{Path: "/foo"},
			},
			expect: true,
		},
		{
			req: &http.Request{
				Host: "bar.example.com",
				URL:  &url.URL{Path: "/bar"},
			},
			expect: true,
		},
		{
			req: &http.Request{
				Host: "foo.example.com",
				URL:  &url.URL{Path: "/baz"},
			},
			expect: false,
		},
		{
			req: &http.Request{
				Host: "baz.example.com",
				URL:  &url.URL{Path: "/foo"},
			},
			expect: false,
		},
	}

	for _, test := range tests {
		_, e, matched := matcher(test.req)

		if matched != test.expect {
			t.Errorf("Matcher for req host %q and path %q = %v; want %v", test.req.Host, test.req.URL.Path, matched, test.expect)
		}

		if matched && e != upstream.Endpoint {
			t.Errorf("unexpected upstream: %s", e)
		}
	}
}
