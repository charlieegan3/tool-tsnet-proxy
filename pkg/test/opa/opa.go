package opatest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/bundle"
)

func NewBundleServer(mods map[string][]byte) (*httptest.Server, error) {
	b := bundle.Bundle{
		Manifest: bundle.Manifest{
			Revision: time.Now().UTC().Format(time.RFC3339),
		},
		Data: map[string]interface{}{},
	}

	for fn, contents := range mods {
		b.Modules = append(b.Modules,
			bundle.ModuleFile{
				URL:    fn,
				Path:   fn,
				Parsed: ast.MustParseModule(string(contents)),
				Raw:    contents,
			},
		)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/vnd.openpolicyagent.bundles")

		err := bundle.NewWriter(w).Write(b)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			//nolint: errcheck
			w.Write([]byte(fmt.Errorf("failed to write bundle: %w", err).Error()))

			return
		}
	})), nil
}
