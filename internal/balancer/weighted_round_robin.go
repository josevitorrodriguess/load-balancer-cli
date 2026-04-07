package balancer

import (
	"log/slog"
	"sync"
)

type WeightedRoundRobin struct {
	mu            sync.Mutex
	backends      []Backend
	current       int
	currentWeight int
	maxWeight     int
	gcdWeight     int
}

func NewWeightedRoundRobin(backends []Backend) *WeightedRoundRobin {
	weighted := &WeightedRoundRobin{
		backends:  backends,
		current:   -1,
		maxWeight: maxBackendWeight(backends),
		gcdWeight: gcdBackendWeight(backends),
	}

	slog.Info("weighted round robin initialized", "backends_count", len(backends))

	return weighted
}

func (wrr *WeightedRoundRobin) NextBackend() (*Backend, error) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	n := len(wrr.backends)
	if n == 0 {
		slog.Error("no backends configured")
		return nil, ErrNoBackendAvailable
	}

	if wrr.maxWeight == 0 {
		slog.Error("no alive backend found")
		return nil, ErrNoBackendAvailable
	}

	for i := 0; i < n*2; i++ {
		wrr.current = (wrr.current + 1) % n
		if wrr.current == 0 {
			wrr.currentWeight -= wrr.gcdWeight
			if wrr.currentWeight <= 0 {
				wrr.currentWeight = wrr.maxWeight
			}
		}

		backend := &wrr.backends[wrr.current]
		if backend.Alive && backend.Weight >= wrr.currentWeight {
			slog.Info("backend selected", "backend_url", backend.URL)
			return backend, nil
		}
	}

	slog.Error("no alive backend found")
	return nil, ErrNoBackendAvailable
}

func (wrr *WeightedRoundRobin) SetBackendAlive(url string, alive bool) error {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	for i := range wrr.backends {
		if wrr.backends[i].URL == url {
			old := wrr.backends[i].Alive
			wrr.backends[i].Alive = alive
			wrr.recalculateWeights()

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

func (wrr *WeightedRoundRobin) IncrementFailCount(url string) error {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	for i := range wrr.backends {
		if wrr.backends[i].URL != url {
			continue
		}

		wrr.backends[i].FailCount++

		if wrr.backends[i].FailCount >= maxFails {
			wrr.backends[i].Alive = false
			wrr.recalculateWeights()

			slog.Warn("backend marked as down",
				"backend_url", url,
				"fail_count", wrr.backends[i].FailCount,
			)
		}

		return nil
	}

	return ErrBackendNotFound
}

func (wrr *WeightedRoundRobin) ResetFailCount(url string) error {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	for i := range wrr.backends {
		if wrr.backends[i].URL != url {
			continue
		}

		wrr.backends[i].FailCount = 0
		return nil
	}

	return ErrBackendNotFound
}

func (wrr *WeightedRoundRobin) Backends() []Backend {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	cloned := make([]Backend, len(wrr.backends))
	copy(cloned, wrr.backends)

	return cloned
}

func (wrr *WeightedRoundRobin) ReportFailure(url string) error {
	return wrr.IncrementFailCount(url)
}

func (wrr *WeightedRoundRobin) IncrementActiveConnections(url string) error {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	for i := range wrr.backends {
		if wrr.backends[i].URL != url {
			continue
		}

		wrr.backends[i].ActiveConnections++
		return nil
	}

	return ErrBackendNotFound
}

func (wrr *WeightedRoundRobin) DecrementActiveConnections(url string) error {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	for i := range wrr.backends {
		if wrr.backends[i].URL != url {
			continue
		}

		if wrr.backends[i].ActiveConnections > 0 {
			wrr.backends[i].ActiveConnections--
		}

		return nil
	}

	return ErrBackendNotFound
}

func (wrr *WeightedRoundRobin) recalculateWeights() {
	wrr.maxWeight = maxBackendWeight(wrr.backends)
	wrr.gcdWeight = gcdBackendWeight(wrr.backends)
	if wrr.currentWeight > wrr.maxWeight {
		wrr.currentWeight = wrr.maxWeight
	}
}

func maxBackendWeight(backends []Backend) int {
	maxWeight := 0

	for _, backend := range backends {
		if !backend.Alive {
			continue
		}

		if backend.Weight > maxWeight {
			maxWeight = backend.Weight
		}
	}

	return maxWeight
}

func gcdBackendWeight(backends []Backend) int {
	current := 0

	for _, backend := range backends {
		if !backend.Alive {
			continue
		}

		if current == 0 {
			current = backend.Weight
			continue
		}

		current = gcd(current, backend.Weight)
	}

	return current
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}

	return a
}
