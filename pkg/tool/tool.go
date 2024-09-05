package tool

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/gorilla/mux"
	"github.com/miekg/dns"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"

	"github.com/charlieegan3/oauth-middleware/pkg/oauthmiddleware"
	"github.com/charlieegan3/tool-tsnet-proxy/pkg/proxy"
	"github.com/charlieegan3/toolbelt/pkg/apis"
)

// Proxy is a tool that runs my reverse proxy
type Proxy struct {
	cfg *proxy.Config

	oauth2Config    *oauth2.Config
	oidcProvider    *oidc.Provider
	idTokenVerifier *oidc.IDTokenVerifier
}

func (p *Proxy) Name() string {
	return "proxy"
}

func (p *Proxy) FeatureSet() apis.FeatureSet {
	return apis.FeatureSet{
		HTTP:     true,
		HTTPHost: true,
		Config:   true,
	}
}

func (p *Proxy) DatabaseMigrations() (*embed.FS, string, error) {
	return nil, "", nil
}

func (p *Proxy) DatabaseSet(db *sql.DB) {
}

func (p *Proxy) SetConfig(config map[string]any) error {
	yamlBS, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	cfg, err := proxy.LoadConfig(bytes.NewReader(yamlBS))
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	p.cfg = cfg

	return nil
}

func (p *Proxy) Jobs() ([]apis.Job, error) { return []apis.Job{}, nil }

func (p *Proxy) HTTPAttach(router *mux.Router) error {
	router.StrictSlash(true)

	var err error
	ctx := context.Background()

	oidcProvider, err := oidc.NewProvider(ctx, p.cfg.OAuth.ProviderURL)
	if err != nil {
		return fmt.Errorf("failed to create oidc provider: %w", err)
	}

	callbackURL, err := url.Parse(p.cfg.OAuth.CallbackURL)
	if err != nil {
		return fmt.Errorf("failed to parse callback url: %w", err)
	}

	oauth2Config := &oauth2.Config{
		Endpoint:     oidcProvider.Endpoint(),
		ClientID:     p.cfg.OAuth.ClientID,
		ClientSecret: p.cfg.OAuth.ClientSecret,
		RedirectURL:  callbackURL.String(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
	}

	idTokenVerifier := oidcProvider.Verifier(
		&oidc.Config{ClientID: oauth2Config.ClientID},
	)

	mw, err := oauthmiddleware.Init(&oauthmiddleware.Config{
		OAuth2Connector: oauth2Config,
		IDTokenVerifier: idTokenVerifier,
		Validators: []oauthmiddleware.IDTokenValidator{
			func(token *oidc.IDToken) (map[any]any, bool) {
				c := struct {
					Email string `json:"email"`
				}{}

				err := token.Claims(&c)
				if err != nil {
					return nil, false
				}

				return map[any]any{"email": c.Email}, true
			},
		},
		AuthBasePath:     "/",
		CallbackBasePath: callbackURL.Path,
	})
	if err != nil {
		return fmt.Errorf("failed to init oauth middleware: %w", err)
	}

	proxyHandler, dnsServers, err := proxy.NewHandlerFromConfig(ctx, p.cfg)
	if err != nil {
		return fmt.Errorf("failed to create proxy: %w", err)
	}

	for _, dnsServer := range dnsServers {
		go func(dnsServer *dns.Server) {
			select {
			case <-ctx.Done():
				shutdownContext, shutdownContextCancel := context.WithTimeout(ctx, 2*time.Second)
				defer shutdownContextCancel()

				if err := dnsServer.ShutdownContext(shutdownContext); err != nil {
					log.Fatalf("Failed to shutdown DNS server: %s\n", err.Error())
				}
			default:
				if err := dnsServer.ListenAndServe(); err != nil {
					log.Fatalf("Failed to start DNS server: %s\n", err.Error())
				}
			}
		}(dnsServer)
	}

	router.Use(mw)
	router.Handle("/", proxyHandler)

	return nil
}

func (p *Proxy) HTTPHost() string {
	hosts := []string{p.cfg.Host}

	for _, upstream := range p.cfg.Upstreams {
		for _, h := range upstream.Hosts {
			hosts = append(hosts, h)
		}
	}

	matcher, err := generateHostRegex(hosts)
	if err != nil {
		log.Fatalf("failed to generate host regex: %v", err)
	}

	return matcher
}
func (p *Proxy) HTTPPath() string { return "" }

func (p *Proxy) ExternalJobsFuncSet(f func(job apis.ExternalJob) error) {}

func generateHostRegex(hosts []string) (string, error) {
	if len(hosts) == 0 {
		return "", errors.New("no hosts provided")
	}

	var rootDomain string
	var subDomains []string

	for _, host := range hosts {
		parsedURL, err := url.Parse(host)
		if err != nil || parsedURL.Host == "" {
			parsedURL = &url.URL{Host: host} // if input is not a URL, treat it as a plain host
		}

		parts := strings.Split(parsedURL.Host, ".")
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid host) %q", host)
		}

		currentRoot := strings.Join(parts[len(parts)-2:], ".")
		if rootDomain == "" {
			rootDomain = currentRoot
		} else if rootDomain != currentRoot {
			return "", errors.New("hosts do not share the same root domain")
		}

		subDomains = append(subDomains, strings.Join(parts[:len(parts)-2], "."))
	}

	pattern := fmt.Sprintf("{%s}.%s", strings.Join(subDomains, "|"), rootDomain)

	return pattern, nil
}
