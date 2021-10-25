package cmd

import (
	"MyDocker/network"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(networkCmd)
	// network create command
	networkCreateCmd.Flags().String("driver", "bridge", "Network driver")
	networkCreateCmd.Flags().String("subnet", "", "subnet cidr")
	networkCmd.AddCommand(networkCreateCmd)
	// network list command
	networkCmd.AddCommand(networkListCmd)
	// network del command
	networkCmd.AddCommand(networkDeleteCmd)
}

// network root command
var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Container network commands",
}

// network subcommand
var networkCreateCmd = &cobra.Command{
	Use:   "create NETWORK [flags]",
	Short: "Create a container network",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing network name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := network.Init(); err != nil {
			return err
		}

		networkName := args[0]
		driver, _ := cmd.Flags().GetString("driver")
		subnet, _ := cmd.Flags().GetString("subnet")

		if err := network.CreateNetwork(driver, subnet, networkName); err != nil {
			return err
		}
		return nil
	},
}

var networkListCmd = &cobra.Command{
	Use:   "list",
	Short: "List container network",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := network.Init(); err != nil {
			return err
		}
		network.ListNetwork()
		return nil
	},
}

var networkDeleteCmd = &cobra.Command{
	Use:   "remove NETWORK",
	Short: "Remove a container network",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing network name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := network.Init(); err != nil {
			return err
		}

		networkName := args[0]
		if err := network.DeleteNetwork(networkName); err != nil {
			return err
		}
		return nil
	},
}
