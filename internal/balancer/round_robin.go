package balancer

import (
	"errors"
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
		return nil, ErrNoBackendAvailable
	}

	for i := 0; i < n; i++ {
		rr.current = (rr.current + 1) % n

		if rr.backends[rr.current].Alive {
			return &rr.backends[rr.current], nil
		}
	}

	return nil, ErrNoBackendAvailable
}

func (rr *RoundRobin) SetBackendAlive(url string, alive bool) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	for i := range rr.backends {
		if rr.backends[i].URL == url {
			rr.backends[i].Alive = alive
			return nil
		}
	}

	return ErrNoBackendAvailable
}