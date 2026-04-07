package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

type StartFunc func(startConfig StartConfig) error

type StartConfig struct {
	ConfigPath string
	DemoMode   bool
}

func Run(args []string, configPath string, startFn StartFunc) error {
	root := newRootCommand(configPath, startFn)
	root.SetArgs(args)
	return root.Execute()
}

func newRootCommand(configPath string, startFn StartFunc) *cobra.Command {
	root := &cobra.Command{
		Use:           "lb",
		Short:         "Control the load balancer via CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.AddCommand(newStartCommand(configPath, startFn))
	root.AddCommand(newConfigCommand(configPath))
	root.AddCommand(newBackendCommand(configPath))
	root.AddCommand(newServerCommand(configPath))

	return root
}

func unavailableStart() error {
	return errors.New("start command is not available")
}
