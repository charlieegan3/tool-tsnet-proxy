package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/open-policy-agent/opa/sdk"

	"github.com/charlieegan3/tool-tsnet-proxy/pkg/opa"
)

type Middleware func(http.Handler) http.Handler

func MiddlewareFromConfigMiddleware(ctx context.Context, config ConfigMiddleware) (Middleware, error) {
	if config.Kind != "opa" || config.OPAProperties == nil {
		return nil, errors.New("invalid config kind")
	}

	bundleServerAddr := config.OPAProperties.Bundle.ServerEndpoint
	bundlePath := config.OPAProperties.Bundle.Path

	opaInstance, err := opa.NewInstance(ctx, opa.InstanceOptions{
		BundleServerAddr: bundleServerAddr,
		BundlePath:       bundlePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OPA instance: %w", err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			statusCode, err := authorizeRequest(ctx, opaInstance, r)
			if err != nil {
				http.Error(w, err.Error(), statusCode)

				return
			}

			next.ServeHTTP(w, r)
		})
	}, nil
}

func authorizeRequest(ctx context.Context, opaInstance *sdk.OPA, r *http.Request) (int, error) {
	rs, err := opaInstance.Decision(ctx, sdk.DecisionOptions{
		Path:  "authz/allow",
		Input: opa.InputFromHTTPRequest(r),
	})
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to get decision: %w", err)
	}

	if rs == nil || rs.Result == nil {
		return http.StatusForbidden, errors.New("not allowed")
	}

	allowed, ok := rs.Result.(bool)
	if !ok || !allowed {
		return http.StatusForbidden, errors.New("not allowed")
	}

	return http.StatusOK, nil
}
