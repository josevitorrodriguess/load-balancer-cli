package health

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/josevitorrodriguess/load-balancer-cli/internal/balancer"
)

type Checker struct {
	balancer balancer.Balancer
	client   *http.Client
	interval time.Duration
}

func New(b balancer.Balancer, interval, timeout time.Duration) *Checker {
	return &Checker{
		balancer: b,
		client: &http.Client{
			Timeout: timeout,
		},
		interval: interval,
	}
}

func (c *Checker) Start() {
	ticker := time.NewTicker(c.interval)

	go func() {
		defer ticker.Stop()

		for range ticker.C {
			c.checkAll()
		}
	}()
}

func (c *Checker) checkAll() {
	backends := c.balancer.Backends()

	for _, backend := range backends {
		c.checkBackend(backend)
	}
}

func (c *Checker) checkBackend(backend balancer.Backend) {
	resp, err := c.client.Get(backend.URL + "/health")
	if err != nil {
		slog.Warn("health check failed",
			"backend_url", backend.URL,
			"error", err,
		)

		if err := c.balancer.ReportFailure(backend.URL); err != nil {
			slog.Error("failed to report backend failure",
				"backend_url", backend.URL,
				"error", err,
			)
		}

		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := c.balancer.ResetFailCount(backend.URL); err != nil {
			slog.Error("failed to reset backend fail count",
				"backend_url", backend.URL,
				"error", err,
			)
			return
		}

		if err := c.balancer.SetBackendAlive(backend.URL, true); err != nil {
			slog.Error("failed to mark backend as up",
				"backend_url", backend.URL,
				"error", err,
			)
			return
		}

		return
	}

	slog.Warn("health check returned unhealthy status",
		"backend_url", backend.URL,
		"status_code", resp.StatusCode,
	)

	if err := c.balancer.ReportFailure(backend.URL); err != nil {
		slog.Error("failed to report backend failure",
			"backend_url", backend.URL,
			"error", err,
		)
	}
}
