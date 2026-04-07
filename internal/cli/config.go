package cli

import (
	"fmt"
	"strings"

	appconfig "github.com/josevitorrodriguess/load-balancer-cli/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCommand(configPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read and update balancer config",
	}

	cmd.AddCommand(newConfigShowCommand(configPath))
	cmd.AddCommand(newConfigSetStrategyCommand(configPath))

	return cmd
}

func newConfigShowCommand(configPath string) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the current config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := appconfig.Load(configPath)
			if err != nil {
				return err
			}

			fmt.Printf("config: %s\n", configPath)
			fmt.Printf("port: %s\n", cfg.Port)
			fmt.Printf("strategy: %s\n", cfg.Strategy)
			fmt.Println("backends:")

			for _, backend := range cfg.Backends {
				fmt.Printf("- url=%s weight=%d\n", backend.URL, backend.Weight)
			}

			fmt.Printf("timeouts: dial=%s tls_handshake=%s response_header=%s health_interval=%s health_timeout=%s\n",
				cfg.Timeouts.Dial.Duration,
				cfg.Timeouts.TLSHandshake.Duration,
				cfg.Timeouts.ResponseHeader.Duration,
				cfg.Timeouts.HealthInterval.Duration,
				cfg.Timeouts.HealthTimeout.Duration,
			)

			return nil
		},
	}
}

func newConfigSetStrategyCommand(configPath string) *cobra.Command {
	return &cobra.Command{
		Use:   "set-strategy <round_robin|weighted_round_robin|least_connections>",
		Short: "Update the load balancing strategy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := appconfig.Load(configPath)
			if err != nil {
				return err
			}

			cfg.Strategy = strings.TrimSpace(args[0])

			if err := appconfig.Save(configPath, cfg); err != nil {
				return err
			}

			fmt.Printf("strategy updated: %s\n", cfg.Strategy)
			return nil
		},
	}
}
