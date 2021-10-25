package cmd

import (
	"MyDocker/container"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "rm NAME",
	Short: "Remove unused container",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing container name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		containerName := args[0]
		return removeContainer(containerName)
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func removeContainer(containerName string) error {
	containerInfo, err := getContainInfoByName(containerName)
	if err != nil {
		return fmt.Errorf("get container info failed: %v", err)
	}
	if containerInfo.Status != container.STOP {
		return fmt.Errorf("couldn't remove running container")
	}

	containerDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(containerDir); err != nil {
		log.Errorf("remove file failed: %v", err)
	}
	container.DeleteWorkSpace(containerInfo.Volume, containerName)
	return nil
}
