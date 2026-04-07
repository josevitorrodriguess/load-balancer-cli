package cli

import (
	"errors"
	"fmt"
	"strings"

	appconfig "github.com/josevitorrodriguess/load-balancer-cli/internal/config"
	"github.com/spf13/cobra"
)

func newBackendCommand(configPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backend",
		Short: "Manage backends",
	}

	cmd.AddCommand(newBackendAddCommand(configPath))
	cmd.AddCommand(newBackendRemoveCommand(configPath))
	cmd.AddCommand(newBackendSetWeightCommand(configPath))

	return cmd
}

func newBackendAddCommand(configPath string) *cobra.Command {
	return &cobra.Command{
		Use:   "add <url> [weight]",
		Short: "Add a backend to the config",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := appconfig.Load(configPath)
			if err != nil {
				return err
			}

			url := strings.TrimSpace(args[0])
			if url == "" {
				return errors.New("backend url is required")
			}

			if backendIndex(cfg, url) >= 0 {
				return fmt.Errorf("backend already exists: %s", url)
			}

			weight := 1
			if len(args) == 2 {
				weight, err = parseWeight(args[1])
				if err != nil {
					return err
				}
			}

			cfg.Backends = append(cfg.Backends, appconfig.Backend{
				URL:    url,
				Weight: weight,
			})

			if err := appconfig.Save(configPath, cfg); err != nil {
				return err
			}

			fmt.Printf("backend added: %s weight=%d\n", url, weight)
			return nil
		},
	}
}

func newBackendRemoveCommand(configPath string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <url>",
		Short: "Remove a backend from the config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := appconfig.Load(configPath)
			if err != nil {
				return err
			}

			url := strings.TrimSpace(args[0])
			index := backendIndex(cfg, url)
			if index < 0 {
				return fmt.Errorf("backend not found: %s", url)
			}

			cfg.Backends = append(cfg.Backends[:index], cfg.Backends[index+1:]...)

			if err := appconfig.Save(configPath, cfg); err != nil {
				return err
			}

			fmt.Printf("backend removed: %s\n", url)
			return nil
		},
	}
}

func newBackendSetWeightCommand(configPath string) *cobra.Command {
	return &cobra.Command{
		Use:   "set-weight <url> <weight>",
		Short: "Update a backend weight",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := appconfig.Load(configPath)
			if err != nil {
				return err
			}

			url := strings.TrimSpace(args[0])
			index := backendIndex(cfg, url)
			if index < 0 {
				return fmt.Errorf("backend not found: %s", url)
			}

			weight, err := parseWeight(args[1])
			if err != nil {
				return err
			}

			cfg.Backends[index].Weight = weight

			if err := appconfig.Save(configPath, cfg); err != nil {
				return err
			}

			fmt.Printf("backend weight updated: %s weight=%d\n", url, weight)
			return nil
		},
	}
}
