package dohtest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

func NewLocalDOHServer(m map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")

		name = strings.TrimSuffix(name, ".")

		if ip, ok := m[name]; ok {
			w.Header().Set("Content-Type", "application/dns-json")
			fmt.Fprintf(w, `{"Status": 0, "Answer": [{"type": 1, "data": "%s"}]}`, ip)
		} else {
			w.Header().Set("Content-Type", "application/dns-json")
			fmt.Fprintf(w, `{"Status": 3}`)
		}
	}))
}
