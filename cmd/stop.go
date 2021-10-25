package cmd

import (
	"MyDocker/container"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop NAME",
	Short: "Stop a container",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing container name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		containerName := args[0]
		return stopContainer(containerName)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func getContainInfoByName(containerName string) (*container.ContainerInfo, error) {
	containerDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := path.Join(containerDir, container.ConfigName)
	content, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("read file %s failed: %v", configFilePath, err)
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(content, &containerInfo); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %v", err)
	}
	return &containerInfo, nil
}

func stopContainer(containerName string) error {
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		return fmt.Errorf("get container pid failed: %v", err)
	}

	pidInt, _ := strconv.Atoi(pid)
	if err := syscall.Kill(pidInt, syscall.SIGTERM); err != nil {
		return fmt.Errorf("stop container failed: %v", err)
	}

	containerInfo, err := getContainInfoByName(containerName)
	if err != nil {
		return fmt.Errorf("get container info failed: %v", err)
	}

	// modify container status
	containerInfo.Status = container.STOP
	containerInfo.PID = " "

	marshalContent, err := json.Marshal(containerInfo)
	if err != nil {
		return fmt.Errorf("marshal container info failed: %v", err)
	}

	containerDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := path.Join(containerDir, container.ConfigName)
	// overwrite
	if err := ioutil.WriteFile(configFilePath, marshalContent, 0622); err != nil {
		return fmt.Errorf("write config file failed: %v", err)
	}
	return nil
}
