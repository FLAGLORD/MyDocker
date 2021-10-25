package cmd

import (
	"MyDocker/container"
	"fmt"
	"os/exec"
	"path"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit CONTAINER IMAGE",
	Short: "commit a container into image",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		containerName := args[0]
		imageName := args[1]
		commitContainer(containerName, imageName)
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
}

func commitContainer(containerName, imageName string) {
	mntURL := fmt.Sprintf(container.MntURL, containerName)
	imageTar := path.Join(container.RootURL, imageName+".tar")
	
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {
		log.Errorf("Tar folder %s error %v", mntURL, err)
	}
}
