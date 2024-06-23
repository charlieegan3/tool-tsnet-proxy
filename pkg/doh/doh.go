package doh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type response struct {
	Status int `json:"Status"`
	Answer []struct {
		Type int    `json:"type"`
		Data string `json:"data"`
	} `json:"Answer"`
}

func QueryA(ctx context.Context, endpoint, domain string) ([]string, error) {
	client := &http.Client{}

	reqURL := fmt.Sprintf("%s?name=%s", endpoint, domain)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Accept", "application/dns-json")

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

	var results []string

	for _, a := range res.Answer {
		if a.Type == 1 {
			results = append(results, a.Data)
		}
	}

	if len(results) == 0 {
		return nil, errors.New("no A records found")
	}

	return results, nil
}
