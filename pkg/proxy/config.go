package proxy

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Addr string `yaml:"addr"`
	Port int    `yaml:"port"`

	DNSServers  []ConfigDNSServer        `yaml:"dns-servers"`
	Middlewares []ConfigMiddleware       `yaml:"middlewares"`
	Upstreams   []ConfigUpstream         `yaml:"upstreams"`
	Tailnets    map[string]ConfigTailnet `yaml:"tailnets"`

	OAuth OAuthConfig `yaml:"oauth"`
}

type ConfigDNSServer struct {
	Addr string `yaml:"addr"`
	Net  string `yaml:"net"`
	DoH  bool   `yaml:"doh"`
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
	Endpoint           string   `yaml:"endpoint"`
	Hosts              []string `yaml:"hosts"`
	PathPrefixes       []string `yaml:"path-prefixes"`
	Tailnet            string   `yaml:"tailnet"`
	InsecureSkipVerify bool     `yaml:"insecure-skip-verify"`
}

type ConfigTailnet struct {
	ID      string `yaml:"id"`
	AuthKey string `yaml:"auth-key"`
}

type OAuthConfig struct {
	CallbackURL  string `yaml:"callback_url"`
	ProviderURL  string `yaml:"provider_url"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

func LoadConfig(r io.Reader) (*Config, error) {
	var cfg Config

	err := yaml.NewDecoder(r).Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &cfg, nil
}
