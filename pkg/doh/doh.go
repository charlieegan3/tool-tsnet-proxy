package doh

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

type Resolver struct {
	Servers []string
}

func (r *Resolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	for _, server := range r.Servers {
		results, err := queryA(ctx, server, host)
		if err != nil {
			return nil, fmt.Errorf("failed to query DoH server: %w", err)
		}

		return results, nil
	}

	return nil, fmt.Errorf("no results found")
}

type response struct {
	Status int `json:"Status"`
	Answer []struct {
		Type int    `json:"type"`
		Data string `json:"data"`
	} `json:"Answer"`
}

func queryA(ctx context.Context, endpoint, domain string) ([]net.IPAddr, error) {
	client := &http.Client{}

	reqURL := fmt.Sprintf("%s?name=%s", endpoint, domain)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("accept", "application/dns-json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var res response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if res.Status != 0 {
		return nil, fmt.Errorf("non-zero status code: %d", res.Status)
	}

	var results []net.IPAddr
	for _, a := range res.Answer {
		if a.Type == 1 {
			ip := net.ParseIP(a.Data)
			if ip == nil {
				continue
			}

			results = append(results, net.IPAddr{IP: ip})
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no A records found")
	}

	return results, nil
}
