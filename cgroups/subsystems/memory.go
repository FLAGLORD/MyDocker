package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

)

type MemorySubsystem struct {
}

func (s *MemorySubsystem) Name() string {
	return "memory"
}

func (s *MemorySubsystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subSystemCgruopPath, err := GetCgroupPath(s.Name(), cgroupPath, true); err == nil {
		if res.MemoryLimit != "" {
			if err := ioutil.WriteFile(path.Join(subSystemCgruopPath, "memory.limit_in_bytes"), []byte(res.MemoryLimit), 0644); err != nil {
				return fmt.Errorf("set cgroup  memory fail %v", err)
			}
		}
		return nil
	} else {
		return err
	}
}

func (s *MemorySubsystem) Remove(cgroupPath string) error {
	if subSystemCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
		return os.Remove(subSystemCgroupPath)
	} else {
		return err
	}
}


func (s *MemorySubsystem) Apply(cgroupPath  string, pid int) error{
	if subSystemCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err != nil{
		if err := ioutil.WriteFile(path.Join(subSystemCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err !=  nil{
			return fmt.Errorf("set cgroup proc fail %v", err)
		}
		return nil
	}else{
		return  fmt.Errorf("get cgroup %s err: %v", cgroupPath, err)
	}
}