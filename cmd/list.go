package cmd

import (
	"MyDocker/container"
	"MyDocker/util"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "ps",
	Short: "List all the containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		return ListContainers()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func ListContainers() error {
	infoDir := path.Dir(fmt.Sprintf(container.DefaultInfoLocation, ""))
	if !util.PathExists(infoDir) {
		os.Mkdir(infoDir, 0666)
	}
	files, err := ioutil.ReadDir(infoDir)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "NAME", "PID", "STATUS", "COMMAND", "CREATED"})
	for _, file := range files {
		if file.Name() == "network" {
			continue
		}
		containerInfo, err := getContainerInfo(file)
		if err != nil {
			return err
		}
		table.Append([]string{containerInfo.ID, containerInfo.Name, containerInfo.PID, containerInfo.Status, containerInfo.Command, containerInfo.CreatedTime})
	}
	table.Render()
	return nil
}

func getContainerInfo(file fs.FileInfo) (*container.ContainerInfo, error) {
	// 获取文件名
	containerName := file.Name()
	configDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configPath := path.Join(configDir, container.ConfigName)

	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read container %s info failed: %v", containerName, err)
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(content, &containerInfo); err != nil {
		return nil, fmt.Errorf("parse container %s info failed: %v", containerName, err)
	}
	return &containerInfo, nil
}
