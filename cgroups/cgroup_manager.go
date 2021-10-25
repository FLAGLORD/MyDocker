package cgroups

import (
	"MyDocker/cgroups/subsystems"
)

type CgroupManager struct {
	Path     string                     // cgroup在hierarchy中的路径，相当于创建的cgroup目录相对于root cgroup的路径
	Resource *subsystems.ResourceConfig //资源配置
}

func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{
		Path: path,
	}
}

// 将进程pid加入到cgroup中
func (c *CgroupManager) Apply(pid int) error {
	for _, subSysIns := range subsystems.SubsystemIns {
		subSysIns.Apply(c.Path, pid)
	}
	return nil
}

// 设置cgroup资源限制
func (c *CgroupManager) Set(res *subsystems.ResourceConfig) error {
	for _, subSysIns := range subsystems.SubsystemIns {
		subSysIns.Set(c.Path, res)
	}
	return nil
}

// release cgroup
func (c *CgroupManager) Destory() error {
	for _, subSysIns := range subsystems.SubsystemIns {
		subSysIns.Remove(c.Path)
	}
	return nil
}
