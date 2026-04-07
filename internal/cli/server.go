package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	appconfig "github.com/josevitorrodriguess/load-balancer-cli/internal/config"
	"github.com/spf13/cobra"
)

func newServerCommand(configPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage server settings",
	}

	cmd.AddCommand(newServerSetPortCommand(configPath))
	cmd.AddCommand(newServerSetTimeoutCommand(configPath))

	return cmd
}

func newServerSetPortCommand(configPath string) *cobra.Command {
	return &cobra.Command{
		Use:   "set-port <port>",
		Short: "Update the server port",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := appconfig.Load(configPath)
			if err != nil {
				return err
			}

			port := strings.TrimSpace(args[0])
			if port == "" {
				return errors.New("port is required")
			}

			cfg.Port = port

			if err := appconfig.Save(configPath, cfg); err != nil {
				return err
			}

			fmt.Printf("server port updated: %s\n", cfg.Port)
			return nil
		},
	}
}

func newServerSetTimeoutCommand(configPath string) *cobra.Command {
	return &cobra.Command{
		Use:   "set-timeout <dial|tls_handshake|response_header|health_interval|health_timeout> <duration>",
		Short: "Update a server timeout value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := appconfig.Load(configPath)
			if err != nil {
				return err
			}

			field := strings.TrimSpace(args[0])
			value, err := time.ParseDuration(strings.TrimSpace(args[1]))
			if err != nil {
				return fmt.Errorf("invalid duration: %s", args[1])
			}

			switch field {
			case "dial":
				cfg.Timeouts.Dial.Duration = value
			case "tls_handshake":
				cfg.Timeouts.TLSHandshake.Duration = value
			case "response_header":
				cfg.Timeouts.ResponseHeader.Duration = value
			case "health_interval":
				cfg.Timeouts.HealthInterval.Duration = value
			case "health_timeout":
				cfg.Timeouts.HealthTimeout.Duration = value
			default:
				return fmt.Errorf("unknown timeout field: %s", field)
			}

			if err := appconfig.Save(configPath, cfg); err != nil {
				return err
			}

			fmt.Printf("server timeout updated: %s=%s\n", field, value)
			return nil
		},
	}
}
