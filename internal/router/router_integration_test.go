//go:build integration

package router

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrwalker511/walk/internal/config"
)

// These tests require llama.cpp running at localhost:8080.
// Run with: go test -tags integration ./internal/router/...

func localCfg() *config.Config {
	return &config.Config{
		LocalModel: config.LocalModel{
			Provider:       "llama.cpp",
			Endpoint:       "http://localhost:8080/v1",
			Model:          "gemma-4-27b-q8_0",
			TimeoutSeconds: 10,
			Enabled:        true,
		},
		Providers: config.Providers{
			Anthropic: config.ProviderConfig{DefaultModel: "claude-sonnet-4-5"},
		},
	}
}

func TestRouteIntegration(t *testing.T) {
	cfg := localCfg()
	r := New(cfg)

	decision, err := r.Route(context.Background(), false)
	require.NoError(t, err)
	assert.Equal(t, DestLocal, decision.Destination)
	assert.Equal(t, cfg.LocalModel.Endpoint, decision.Endpoint)
	t.Logf("Routed to local: model=%s reason=%s", decision.Model, decision.Reason)
}

func TestCheckLocalHealthIntegration(t *testing.T) {
	cfg := localCfg()
	r := New(cfg)

	healthy, err := r.CheckLocalHealth(context.Background())
	require.NoError(t, err)
	assert.True(t, healthy, "expected llama.cpp at localhost:8080 to be healthy")
}
