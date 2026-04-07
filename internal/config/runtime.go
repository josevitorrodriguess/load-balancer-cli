package config

import (
	"fmt"

	"github.com/josevitorrodriguess/load-balancer-cli/internal/balancer"
	"github.com/josevitorrodriguess/load-balancer-cli/internal/proxy"
)

func NewBalancer(cfg Config) (balancer.Balancer, error) {
	return NewBalancerWithState(cfg, nil)
}

func NewBalancerWithState(cfg Config, previous []balancer.Backend) (balancer.Balancer, error) {
	state := make(map[string]balancer.Backend, len(previous))
	for _, backend := range previous {
		state[backend.URL] = backend
	}

	backends := make([]balancer.Backend, 0, len(cfg.Backends))
	for _, backendCfg := range cfg.Backends {
		backend := balancer.Backend{
			URL:    backendCfg.URL,
			Alive:  true,
			Weight: backendCfg.Weight,
		}

		if existing, ok := state[backendCfg.URL]; ok {
			backend.Alive = existing.Alive
			backend.FailCount = existing.FailCount
			backend.ActiveConnections = existing.ActiveConnections
		}

		backends = append(backends, balancer.Backend{
			URL:       backend.URL,
			Alive:     backend.Alive,
			FailCount: backend.FailCount,
			Weight:    backend.Weight,
			ActiveConnections: backend.ActiveConnections,
		})
	}

	switch cfg.Strategy {
	case DefaultStrategy:
		return balancer.NewRoundRobin(backends), nil
	case WeightedRoundRobinStrategy:
		return balancer.NewWeightedRoundRobin(backends), nil
	case LeastConnectionsStrategy:
		return balancer.NewLeastConnections(backends), nil
	default:
		return nil, fmt.Errorf("unsupported strategy: %s", cfg.Strategy)
	}
}

func ProxyConfig(cfg Config) proxy.Config {
	return proxy.Config{
		DialTimeout:           cfg.Timeouts.Dial.Duration,
		TLSHandshakeTimeout:   cfg.Timeouts.TLSHandshake.Duration,
		ResponseHeaderTimeout: cfg.Timeouts.ResponseHeader.Duration,
	}
}
