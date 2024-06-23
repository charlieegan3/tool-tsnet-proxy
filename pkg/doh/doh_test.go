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
	t.Parallel()

	testDoHServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	results, err := QueryA(context.Background(), testDoHServer.URL, "example.com")
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	exp := "93.184.215.14"
	if results[0] != exp {
		t.Fatalf("Expected %s, got %s", exp, results[0])
	}
}
