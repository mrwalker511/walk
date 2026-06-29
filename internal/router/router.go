package router

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mrwalker511/walk/internal/config"
)

// Destination represents the chosen routing target.
type Destination string

const (
	DestLocal Destination = "local"  // llama.cpp
	DestCloud Destination = "cloud"  // Anthropic / OpenAI
)

// Decision is the output of a routing decision.
type Decision struct {
	Destination Destination
	Endpoint    string
	Model       string
	Reason      string
}

// Client is an HTTP client interface for testability.
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// Router decides where to send a request based on config and live health checks.
type Router struct {
	cfg    *config.Config
	client Client
}

// New creates a Router with the given config and a default HTTP client.
func New(cfg *config.Config) *Router {
	return &Router{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.LocalModel.TimeoutSeconds) * time.Second,
		},
	}
}

// NewWithClient creates a Router with a custom HTTP client (for testing).
func NewWithClient(cfg *config.Config, client Client) *Router {
	return &Router{cfg: cfg, client: client}
}

// Route decides where to send the request.
// If local model is enabled and healthy, routes locally; otherwise falls back to cloud.
func (r *Router) Route(ctx context.Context, forceLocal bool) (Decision, error) {
	if !r.cfg.LocalModel.Enabled && !forceLocal {
		return r.cloudDecision("local model disabled in config"), nil
	}

	healthy, err := r.checkLocalHealth(ctx)
	if err != nil || !healthy {
		if forceLocal {
			return Decision{}, fmt.Errorf("local llama.cpp is not available at %s (hint: start with 'llama-server --model /path/to/model.gguf --port 8080')", r.cfg.LocalModel.Endpoint)
		}
		return r.cloudDecision(fmt.Sprintf("llama.cpp health check failed: %v", err)), nil
	}

	return Decision{
		Destination: DestLocal,
		Endpoint:    r.cfg.LocalModel.Endpoint,
		Model:       r.cfg.LocalModel.Model,
		Reason:      "llama.cpp is healthy",
	}, nil
}

// CheckLocalHealth returns true if the llama.cpp health endpoint responds OK.
func (r *Router) CheckLocalHealth(ctx context.Context) (bool, error) {
	return r.checkLocalHealth(ctx)
}

func (r *Router) checkLocalHealth(ctx context.Context) (bool, error) {
	healthURL := r.cfg.LocalModel.Endpoint
	// Strip /v1 suffix to get base URL for /health
	base := healthURL
	for _, suffix := range []string{"/v1", "/v1/"} {
		if len(base) > len(suffix) && base[len(base)-len(suffix):] == suffix {
			base = base[:len(base)-len(suffix)]
		}
	}
	healthURL = base + "/health"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false, fmt.Errorf("building health request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("health check: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK, nil
}

func (r *Router) cloudDecision(reason string) Decision {
	model := r.cfg.Providers.Anthropic.DefaultModel
	if model == "" {
		model = "claude-sonnet-4-5"
	}
	return Decision{
		Destination: DestCloud,
		Endpoint:    "https://api.anthropic.com/v1",
		Model:       model,
		Reason:      reason,
	}
}
