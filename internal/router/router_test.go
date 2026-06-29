package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mrwalker511/walk/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig(endpoint string, enabled bool) *config.Config {
	return &config.Config{
		LocalModel: config.LocalModel{
			Provider:       "llama.cpp",
			Endpoint:       endpoint,
			Model:          "gemma-4-27b-q8_0",
			TimeoutSeconds: 5,
			Enabled:        enabled,
		},
		Providers: config.Providers{
			Anthropic: config.ProviderConfig{
				DefaultModel: "claude-sonnet-4-5",
			},
		},
	}
}

func TestRouteToLocalWhenHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL+"/v1", true)
	router := NewWithClient(cfg, srv.Client())

	decision, err := router.Route(context.Background(), false)
	require.NoError(t, err)
	assert.Equal(t, DestLocal, decision.Destination)
	assert.Equal(t, srv.URL+"/v1", decision.Endpoint)
	assert.Equal(t, "gemma-4-27b-q8_0", decision.Model)
}

func TestRouteToCloudWhenLocalUnhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL+"/v1", true)
	router := NewWithClient(cfg, srv.Client())

	decision, err := router.Route(context.Background(), false)
	require.NoError(t, err)
	assert.Equal(t, DestCloud, decision.Destination)
	assert.Equal(t, "claude-sonnet-4-5", decision.Model)
}

func TestRouteToCloudWhenLocalDisabled(t *testing.T) {
	cfg := testConfig("http://localhost:8080/v1", false)
	router := New(cfg)

	decision, err := router.Route(context.Background(), false)
	require.NoError(t, err)
	assert.Equal(t, DestCloud, decision.Destination)
	assert.Contains(t, decision.Reason, "disabled")
}

func TestForceLocalFailsWhenUnhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL+"/v1", true)
	router := NewWithClient(cfg, srv.Client())

	_, err := router.Route(context.Background(), true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "llama.cpp is not available")
	assert.Contains(t, err.Error(), "hint:")
}

func TestHealthCheckURLStripping(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL+"/v1", true)
	router := NewWithClient(cfg, srv.Client())

	_, err := router.CheckLocalHealth(context.Background())
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(capturedPath, "/health"), "expected /health path, got %s", capturedPath)
}
