package cmd

import (
	"MyDocker/cgroups"
	"MyDocker/cgroups/subsystems"
	"MyDocker/container"
	"MyDocker/network"
	"MyDocker/reexec"
	"MyDocker/util"
	"fmt"
	"strconv"

	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolP("interactive", "i", false, "keep STDIN open even if not attached")
	runCmd.Flags().BoolP("tty", "t", false, "Allocate a pesudo TTY")
	runCmd.Flags().StringP("volume", "v", "", "Data volume")
	runCmd.Flags().BoolP("detach", "d", false, "Detach container")
	runCmd.Flags().String("name", "", "Container name")
	runCmd.Flags().StringSliceP("environment", "e", []string{}, "Environment Set")
	runCmd.Flags().String("net", "", "Container network")
	runCmd.Flags().StringSliceP("port", "p", []string{}, "Port mapping")
	// resources limit
	runCmd.Flags().StringP("memory", "m", "", "Memory limit")
	runCmd.Flags().String("cpushare", "", "CPUshare limit")
	runCmd.Flags().String("cpuset", "", "CPUset limit")
	reexec.Register("containerInitProcess", container.RunContainerInitProcess)
	if reexec.Init() {
		os.Exit(0)
	}
}

var runCmd = &cobra.Command{
	Use:   "run IMAGE COMMAND",
	Short: "Create a container with namespace andd cgroups limit",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing container image")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		tty, _ := cmd.Flags().GetBool("tty")
		volume, _ := cmd.Flags().GetString("volume")
		detach, _ := cmd.Flags().GetBool("detach")
		name, _ := cmd.Flags().GetString("name")
		network, _ := cmd.Flags().GetString("net")
		envSlice, _ := cmd.Flags().GetStringSlice("environment")
		portmapping, _ := cmd.Flags().GetStringSlice("port")

		memoryLimit, _ := cmd.Flags().GetString("memory")
		cpuShare, _ := cmd.Flags().GetString("cpushare")
		cpuSet, _ := cmd.Flags().GetString("cpuset")

		cmd.SilenceUsage = true
		// detach 和 tty 无法共存
		if detach && tty {
			return fmt.Errorf("for interactive process, it should not be detached")
		}

		resConf := &subsystems.ResourceConfig{
			MemoryLimit: memoryLimit,
			CpuShare:    cpuShare,
			CpuSet:      cpuSet,
		}

		return run(tty, volume, name, network, args, portmapping, envSlice, resConf)
	},
}

func run(tty bool, volume string, containerName string, nw string, cmdArray []string, portmapping []string, envSlice []string, res *subsystems.ResourceConfig) error {
	// generate container ID
	containerID, err := util.RandString(10)
	if err != nil {
		return fmt.Errorf("generate containerID failed: %v", err)
	}
	if containerName == "" {
		containerName = containerID
	}

	// create container porcess
	imageName := cmdArray[0]
	containerProcess, writePipe, err := container.NewContainerProcess(tty, containerName, volume, imageName, envSlice)
	if err != nil {
		return err
	}
	if err := containerProcess.Start(); err != nil {
		log.Error(err)
	}

	// record container info
	err = container.RecordContainerInfo(containerProcess.Process.Pid, containerName, containerID, volume, cmdArray)
	if err != nil {
		return err
	}

	// use containerID  as cgroup name
	cgroupManager := cgroups.NewCgroupManager(containerID)
	defer cgroupManager.Destory()
	cgroupManager.Set(res)
	cgroupManager.Apply(containerProcess.Process.Pid)

	if nw != "" {
		// config network
		if err := network.Init(); err != nil {
			return err
		}
		containerInfo := &container.ContainerInfo{
			ID:          containerID,
			PID:         strconv.Itoa(containerProcess.Process.Pid),
			Name:        containerName,
			PortMapping: portmapping,
		}
		if err := network.Connect(nw, containerInfo); err != nil {
			return fmt.Errorf("config network failed: %v", err)
		}
	}

	sendInitCommand(cmdArray, writePipe)
	if tty {
		containerProcess.Wait()
		if err := container.DeleteContainerInfo(containerName); err != nil {
			return err
		}
		container.DeleteWorkSpace(volume, containerName)
	}
	return nil
}

func sendInitCommand(cmdArray []string, writePipe *os.File) {
	command := strings.Join(cmdArray, " ")
	writePipe.WriteString(command)
	writePipe.Close()
}
