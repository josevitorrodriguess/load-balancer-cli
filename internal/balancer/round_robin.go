package balancer

import (
	"errors"
	"log/slog"
	"sync"
)

var ErrNoBackendAvailable = errors.New("no backend available")

type Backend struct {
	URL   string
	Alive bool
}

type RoundRobin struct {
	mu       sync.Mutex
	backends []Backend
	current  int
}

func NewRoundRobin(backends []Backend) *RoundRobin {
	slog.Info("round robin initialized",
		"backends_count", len(backends),
	)

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

	slog.Error("backend not found",
		"backend_url", url,
	)

	return ErrNoBackendAvailable
}