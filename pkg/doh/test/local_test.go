package doh_test

import (
	"testing"

	"github.com/charlieegan3/tool-tsnet-proxy/pkg/doh"
)

func TestNewLocalDOHServer(t *testing.T) {

	dohServer := NewLocalDOHServer(
		map[string]string{
			"example.com": "0.0.0.0",
		},
	)
	defer dohServer.Close()

	results, err := doh.QueryA(dohServer.URL, "example.com")
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	exp := "0.0.0.0"
	if results[0] != exp {
		t.Fatalf("Expected %s, got %s", exp, results[0])
	}
}
