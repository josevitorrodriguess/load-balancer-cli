package balancer

import (
	"errors"
	"log/slog"
	"sync"
)

var ErrNoBackendAvailable = errors.New("no backend available")
var ErrBackendNotFound = errors.New("backend not found")

const maxFails = 3

type Backend struct {
	URL       string
	Alive     bool
	FailCount int
}

type RoundRobin struct {
	mu       sync.Mutex
	backends []Backend
	current  int
}

func NewRoundRobin(backends []Backend) *RoundRobin {
	slog.Info("round robin initialized", "backends_count", len(backends))

	return &RoundRobin{
		backends: backends,
		current:  -1,
	}
}

func (rr *RoundRobin) NextBackend() (*Backend, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	n := len(rr.backends)
	if n == 0 {
		slog.Error("no backends configured")
		return nil, ErrNoBackendAvailable
	}

	for i := 0; i < n; i++ {
		rr.current = (rr.current + 1) % n
		backend := rr.backends[rr.current]

		if backend.Alive {
			slog.Info("backend selected",
				"backend_url", backend.URL,
				"index", rr.current,
			)
			return &rr.backends[rr.current], nil
		}

		slog.Warn("skipping dead backend",
			"backend_url", backend.URL,
			"index", rr.current,
		)
	}

	slog.Error("no alive backend found")
	return nil, ErrNoBackendAvailable
}

func (rr *RoundRobin) SetBackendAlive(url string, alive bool) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	for i := range rr.backends {
		if rr.backends[i].URL == url {
			old := rr.backends[i].Alive
			rr.backends[i].Alive = alive

			slog.Info("backend health changed",
				"backend_url", url,
				"old_status", old,
				"new_status", alive,
			)

			return nil
		}
	}

	slog.Error("backend not found", "backend_url", url)
	return ErrBackendNotFound
}

func (rr *RoundRobin) IncrementFailCount(url string) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	for i := range rr.backends {
		if rr.backends[i].URL != url {
			continue
		}

		rr.backends[i].FailCount++

		slog.Warn("backend failure recorded",
			"backend_url", url,
			"fail_count", rr.backends[i].FailCount,
		)

		if rr.backends[i].FailCount >= maxFails {
			rr.backends[i].Alive = false

			slog.Warn("backend marked as down",
				"backend_url", url,
				"fail_count", rr.backends[i].FailCount,
			)
		}

		return nil
	}

	return ErrBackendNotFound
}

func (rr *RoundRobin) ResetFailCount(url string) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	for i := range rr.backends {
		if rr.backends[i].URL != url {
			continue
		}

		if rr.backends[i].FailCount != 0 {
			slog.Info("backend fail count reset",
				"backend_url", url,
				"old_fail_count", rr.backends[i].FailCount,
			)
		}

		rr.backends[i].FailCount = 0
		return nil
	}

	return ErrBackendNotFound
}

func (rr *RoundRobin) Backends() []Backend {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	cloned := make([]Backend, len(rr.backends))
	copy(cloned, rr.backends)

	return cloned
}

func (rr *RoundRobin) ReportFailure(url string) error {
	return rr.IncrementFailCount(url)
}