package balancer

import (
	"log/slog"
	"sync"
)

type LeastConnections struct {
	mu       sync.Mutex
	backends []Backend
	current  int
}

func NewLeastConnections(backends []Backend) *LeastConnections {
	slog.Info("least connections initialized", "backends_count", len(backends))

	return &LeastConnections{
		backends: backends,
		current:  -1,
	}
}

func (lc *LeastConnections) NextBackend() (*Backend, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if len(lc.backends) == 0 {
		slog.Error("no backends configured")
		return nil, ErrNoBackendAvailable
	}

	minConnections := -1
	for i := range lc.backends {
		if !lc.backends[i].Alive {
			continue
		}

		if minConnections == -1 || lc.backends[i].ActiveConnections < minConnections {
			minConnections = lc.backends[i].ActiveConnections
		}
	}

	if minConnections == -1 {
		slog.Error("no alive backend found")
		return nil, ErrNoBackendAvailable
	}

	selectedIndex := -1
	for offset := 1; offset <= len(lc.backends); offset++ {
		index := (lc.current + offset) % len(lc.backends)
		backend := lc.backends[index]
		if !backend.Alive {
			continue
		}

		if backend.ActiveConnections == minConnections {
			selectedIndex = index
			break
		}
	}

	if selectedIndex == -1 {
		slog.Error("no alive backend found")
		return nil, ErrNoBackendAvailable
	}

	lc.current = selectedIndex

	slog.Info("backend selected", "backend_url", lc.backends[selectedIndex].URL)
	return &lc.backends[selectedIndex], nil
}

func (lc *LeastConnections) SetBackendAlive(url string, alive bool) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	for i := range lc.backends {
		if lc.backends[i].URL == url {
			old := lc.backends[i].Alive
			lc.backends[i].Alive = alive

			if old != alive {
				slog.Info("backend health changed",
					"backend_url", url,
					"alive", alive,
				)
			}

			return nil
		}
	}

	slog.Error("backend not found", "backend_url", url)
	return ErrBackendNotFound
}

func (lc *LeastConnections) IncrementFailCount(url string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	for i := range lc.backends {
		if lc.backends[i].URL != url {
			continue
		}

		lc.backends[i].FailCount++

		if lc.backends[i].FailCount >= maxFails {
			lc.backends[i].Alive = false

			slog.Warn("backend marked as down",
				"backend_url", url,
				"fail_count", lc.backends[i].FailCount,
			)
		}

		return nil
	}

	return ErrBackendNotFound
}

func (lc *LeastConnections) ResetFailCount(url string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	for i := range lc.backends {
		if lc.backends[i].URL != url {
			continue
		}

		lc.backends[i].FailCount = 0
		return nil
	}

	return ErrBackendNotFound
}

func (lc *LeastConnections) Backends() []Backend {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	cloned := make([]Backend, len(lc.backends))
	copy(cloned, lc.backends)

	return cloned
}

func (lc *LeastConnections) ReportFailure(url string) error {
	return lc.IncrementFailCount(url)
}

func (lc *LeastConnections) IncrementActiveConnections(url string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	for i := range lc.backends {
		if lc.backends[i].URL != url {
			continue
		}

		lc.backends[i].ActiveConnections++
		return nil
	}

	return ErrBackendNotFound
}

func (lc *LeastConnections) DecrementActiveConnections(url string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	for i := range lc.backends {
		if lc.backends[i].URL != url {
			continue
		}

		if lc.backends[i].ActiveConnections > 0 {
			lc.backends[i].ActiveConnections--
		}

		return nil
	}

	return ErrBackendNotFound
}
