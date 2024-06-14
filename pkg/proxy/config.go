package proxy

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v2"
)

type Config struct {
	DNSServers  []string           `yaml:"dns-servers"`
	Middlewares []ConfigMiddleware `yaml:"middlewares"`
	Upstreams   []ConfigUpstream   `yaml:"upstreams"`
}

type ConfigMiddleware struct {
	Kind          string                    `yaml:"kind"`
	OPAProperties *ConfigMiddlewarePropsOPA `yaml:"properties"`
}

type ConfigMiddlewarePropsOPA struct {
	Bundle ConfigMiddlewarePropsOPABundle `yaml:"bundle"`
}

type ConfigMiddlewarePropsOPABundle struct {
	ServerEndpoint string `yaml:"server-endpoint"`
	Path           string `yaml:"path"`
}

type ConfigUpstream struct {
	Endpoint     string   `yaml:"endpoint"`
	Hosts        []string `yaml:"hosts"`
	PathPrefixes []string `yaml:"path-prefixes"`
}

func LoadConfig(r io.Reader) (*Config, error) {
	var cfg Config

	err := yaml.NewDecoder(r).Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &cfg, nil
}
