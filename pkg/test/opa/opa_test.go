package opatest

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/open-policy-agent/opa/sdk"

	"github.com/charlieegan3/tool-tsnet-proxy/pkg/opa"
)

func TestNewBundlerServer(t *testing.T) {
	t.Parallel()

	files := map[string][]byte{
		"example.rego": []byte(`package example

allow := "foo"`),
	}

	bs, err := NewBundleServer(files)
	if err != nil {
		t.Fatalf("Failed to create bundle server: %v", err)
	}
	defer bs.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	opaInstance, err := opa.NewInstance(ctx, opa.InstanceOptions{
		BundleServerAddr: bs.URL,
		BundlePath:       "bundle.tar.gz",
	})
	if err != nil {
		t.Fatalf("Failed to create OPA instance: %v", err)
	}

	rs, err := opaInstance.Decision(ctx, sdk.DecisionOptions{
		Path: "example/allow",
	})
	if err != nil {
		t.Fatalf("Failed to get decision: %v", err)
	}

	if rs == nil {
		t.Fatalf("Decision was nil")
	}

	if rs.Result == nil {
		t.Fatalf("Decision result was nil")
	}

	drBs, err := json.Marshal(rs.Result)
	if err != nil {
		t.Fatalf("Failed to marshal decision result: %v", err)
	}

	if string(drBs) != `"foo"` {
		t.Fatalf("Unexpected decision result: %s", string(drBs))
	}
}
