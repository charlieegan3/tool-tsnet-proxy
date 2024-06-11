package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	utils_test "github.com/charlieegan3/tool-tsnet-proxy/pkg/test/utils"
)

func TestSimpleUseCase(t *testing.T) {

	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, client")
	}))

	backendServerURL, err := url.Parse(backendServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	dohServer := doh_test.NewLocalDOHServer(
		map[string]string{
			"example.com": "127.0.0.1",
		},
	)
	defer dohServer.Close()

	dohServerURL := dohServer.URL

	opts := &Options{
		DoHURL: dohServerURL,
		PortMappings: map[string]string{
			"example.com": backendServerURL.Port(),
		},
	}

	port := utils_test.FreePort(0)
	if err != nil {
		t.Fatal(err)
	}

	addr := fmt.Sprintf("localhost:%d", port)

	p, err := New(addr, opts)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/foobar", addr), nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Host", "example.com")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "Hello, client" {
		t.Fatalf("Expected 'Hello, client', got %s", string(body))
	}
}
