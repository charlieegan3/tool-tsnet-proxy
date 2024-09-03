package proxy

import (
	"net/http"
	"strings"
)

type Matcher func(*http.Request) (*http.Client, string, bool)

func MatcherFromUpstream(upstream ConfigUpstream, client *http.Client) Matcher {
	return func(req *http.Request) (*http.Client, string, bool) {
		if !matchesHost(req.Host, upstream.Hosts) {
			return nil, "", false
		}

		if !matchesPath(req.URL.Path, upstream.PathPrefixes) {
			return nil, "", false
		}

		return client, upstream.Endpoint, true
	}
}

func matchesHost(host string, hosts []string) bool {
	if len(hosts) == 0 {
		return true
	}

	for _, h := range hosts {
		if strings.HasPrefix(host, h) {
			return true
		}
	}

	return false
}

func matchesPath(path string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}
