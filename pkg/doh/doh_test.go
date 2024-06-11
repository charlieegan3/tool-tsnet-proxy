package doh

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestQuery(t *testing.T) {
	testDoHServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("fixtures/resp.json")
		if err != nil {
			t.Fatal(err)
		}

		_, err = io.Copy(w, file)
		if err != nil {
			t.Fatal(err)
		}
	}))

	defer testDoHServer.Close()

	res := Resolver{
		Servers: []string{testDoHServer.URL},
	}

	results, err := res.LookupIPAddr(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	exp := "93.184.215.14"
	if results[0].String() != exp {
		t.Fatalf("Expected %s, got %s", exp, results[0].String())
	}
}
