package cli

import "github.com/spf13/cobra"

func newStartCommand(configPath string, startFn StartFunc) *cobra.Command {
	var demo bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the load balancer",
		RunE: func(cmd *cobra.Command, args []string) error {
			if startFn == nil {
				return unavailableStart()
			}

			return startFn(StartConfig{
				ConfigPath: configPath,
				DemoMode:   demo,
			})
		},
	}

	cmd.Flags().BoolVar(&demo, "demo", false, "start demo backends before the load balancer")

	return cmd
}
