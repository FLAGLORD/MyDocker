package container

import (
	"MyDocker/reexec"
	"MyDocker/util"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	RUNNING             string = "running"
	STOP                string = "stopped"
	EXIT                string = "exited"
	DefaultInfoLocation string = "/var/run/mydocker/%s/"
	ConfigName          string = "config.json"
	ContainerLogFile    string = "container.log"
)

type ContainerInfo struct {
	PID         string   `json:"pid"`         // 容器进程在 host 上的 pid
	ID          string   `json:"id"`          // 容器 ID
	Name        string   `json:"name"`        // 容器名
	Command     string   `json:"command"`     // 容器进程运行的命令
	CreatedTime string   `json:"createdTime"` // 创建时间
	Status      string   `json:"status"`      // 容器状态
	Volume      string   `json:"volume"`      // 数据卷
	PortMapping []string `json:"portmapping"` // 端口映射
}

func RecordContainerInfo(containerPID int, containerName string, containerID string, volume string, cmdArray []string) error {

	createdTime := time.Now().Format("2006-01-02 15:04:59")
	command := strings.Join(cmdArray, " ")

	containerInfo := &ContainerInfo{
		PID:         strconv.Itoa(containerPID),
		ID:          containerID,
		Command:     command,
		CreatedTime: createdTime,
		Status:      RUNNING,
		Name:        containerName,
		Volume:      volume,
	}

	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		return fmt.Errorf("record container info: %v", err)
	}
	infoStorageDir := getContainerInfoDir(containerName)
	if !util.PathExists(infoStorageDir) {
		// rw-r-r
		if err := os.MkdirAll(infoStorageDir, 0666); err != nil {
			return fmt.Errorf("record container info: %v", err)
		}
	}

	configFileName := path.Join(infoStorageDir, ConfigName)
	configFile, err := os.Create(configFileName)
	if err != nil {
		return fmt.Errorf("record container info: %v", err)
	}
	defer configFile.Close()
	// write json to file
	if _, err := configFile.WriteString(string(jsonBytes)); err != nil {
		return fmt.Errorf("record container info: %v", err)
	}

	return nil
}

func DeleteContainerInfo(containerName string) error {
	containerInfoDir := getContainerInfoDir(containerName)
	if err := os.RemoveAll(containerInfoDir); err != nil {
		return fmt.Errorf("delete container info: %v", err)
	}
	return nil
}

// NewContainerProcess creates a container process
func NewContainerProcess(tty bool, containerName string, volume string, imageName string, envSlice []string) (*exec.Cmd, *os.File, error) {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return nil, nil, fmt.Errorf("new pipe failed: %v", err)
	}
	cmd := reexec.Command("containerInitProcess")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUSER | syscall.CLONE_NEWNET,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      syscall.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      syscall.Getgid(),
				Size:        1,
			},
		},
	}

	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		containerInfoDir := getContainerInfoDir(containerName)
		if !util.PathExists(containerInfoDir) {
			if err := os.MkdirAll(containerInfoDir, 0622); err != nil {
				log.Error(err)
			}
		}
		logFilePath := path.Join(containerInfoDir, ContainerLogFile)
		logFileFd, err := os.Create(logFilePath)
		if err != nil {
			return nil, nil, err
		}
		cmd.Stdout = logFileFd
		cmd.Stderr = logFileFd
	}
	cmd.ExtraFiles = []*os.File{readPipe}
	cmd.Env = append(os.Environ(), envSlice...)
	if err := NewWorkSpace(volume, imageName, containerName); err != nil {
		return nil, nil, err
	}
	cmd.Dir = fmt.Sprintf(MntURL, containerName)
	return cmd, writePipe, nil
}

// getContainerInfoDir returns url where container infomation is stored
func getContainerInfoDir(containerName string) string {
	return fmt.Sprintf(DefaultInfoLocation, containerName)
}
