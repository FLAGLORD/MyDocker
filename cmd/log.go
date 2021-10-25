package cmd

import (
	"MyDocker/container"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(logCmd)
}

var logCmd = &cobra.Command{
	Use:   "logs NAME",
	Short: "Print logs of a container",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("Need a container name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		containerName := args[0]
		cmd.SilenceUsage = true
		return logContainer(containerName)
	},
}

func logContainer(containerName string) error {
	containerURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	logFilePath := path.Join(containerURL, container.ContainerLogFile)
	file, err := os.Open(logFilePath)
	if err != nil {
		return fmt.Errorf("log container open file %s failed: %v", logFilePath, err)
	}

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("log container read file %s failed: %v", logFilePath, err)
	}
	fmt.Fprintf(os.Stdout, string(content))
	return nil
}
