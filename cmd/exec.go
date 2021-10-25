package cmd

import (
	"MyDocker/container"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	_ "MyDocker/nsenter"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const ENV_EXEC_PID = "mydocker_pid"
const ENv_EXEC_CMD = "mydocker_cmd"

var execCmd = &cobra.Command{
	Use:   "exec NAME COMMAND",
	Short: "Exec a command into container",
	RunE: func(cmd *cobra.Command, args []string) error {

		if os.Getenv(ENV_EXEC_PID) != "" {
			log.Infof("pid callback pid %s", os.Getgid())
		}

		if len(args) < 2 {
			return fmt.Errorf("missing container name or command")
		}

		containerName := args[0]

		return ExecContainer(containerName, args[1:])
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}

func getContainerPidByName(containerName string) (string, error) {
	containerDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := path.Join(containerDir, container.ConfigName)

	content, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return "", err
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(content, &containerInfo); err != nil {
		return "", err
	}
	return containerInfo.PID, nil
}

func getEnvsByPid(pid string) ([]string, error) {
	path := fmt.Sprintf("/proc/%s/environ", pid)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("getEnvsByPid get process %s environ failed: %v", pid, err)
	}
	// 多个环境变量中的分隔符是 \u0000
	envs := strings.Split(string(content), "\u0000")
	return envs, nil
}

func ExecContainer(containerName string, cmdArray []string) error {
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		return fmt.Errorf("exeContainer getContainerPidByName failed: %v", err)
	}

	cmdStr := strings.Join(cmdArray, " ")
	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	os.Setenv(ENV_EXEC_PID, pid)
	os.Setenv(ENv_EXEC_CMD, cmdStr)

	containerEnvs, err := getEnvsByPid(pid)
	if  err != nil{
		return err
	}
	cmd.Env = append(os.Environ(), containerEnvs...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("execContainer %s failed: %v", containerName, err)
	}
	return nil
}
