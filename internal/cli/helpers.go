package cli

import (
	"errors"
	"fmt"
	"strconv"

	appconfig "github.com/josevitorrodriguess/load-balancer-cli/internal/config"
)

func backendIndex(cfg appconfig.Config, url string) int {
	for i, backend := range cfg.Backends {
		if backend.URL == url {
			return i
		}
	}

	return -1
}

func parseWeight(value string) (int, error) {
	weight, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid weight: %s", value)
	}

	if weight <= 0 {
		return 0, errors.New("weight must be greater than zero")
	}

	return weight, nil
}
