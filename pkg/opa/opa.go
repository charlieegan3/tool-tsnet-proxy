package opa

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/logging"
	"github.com/open-policy-agent/opa/sdk"
	"gopkg.in/yaml.v2"
)

type InstanceOptions struct {
	BundleServerAddr string
	BundlePath       string
}

func NewInstance(ctx context.Context, opts InstanceOptions) (*sdk.OPA, error) {
	cfg := struct {
		Services map[string]map[string]interface{} `yaml:"services"`
		Bundles  map[string]map[string]interface{} `yaml:"bundles"`
	}{
		Services: map[string]map[string]interface{}{
			"main": {
				"url": opts.BundleServerAddr,
			},
		},
		Bundles: map[string]map[string]interface{}{
			"main": {
				"service":  "main",
				"resource": "/" + strings.TrimPrefix(opts.BundlePath, "/"),
			},
		},
	}

	cfgBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	inst, err := sdk.New(ctx, sdk.Options{
		Config: bytes.NewReader(cfgBytes),
		Logger: logging.New(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OPA instance: %w", err)
	}

	return inst, nil
}
