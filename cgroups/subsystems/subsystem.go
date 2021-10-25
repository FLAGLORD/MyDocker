package subsystems


type ResourceConfig struct{
	MemoryLimit string
	CpuShare string	// CPU timeslice weight
	CpuSet string	// CPU cors
}

type Subsystem interface{
	Name() string	// return the name of subsystem, for example, cpu memory 
	Set(path string,  res *ResourceConfig) error	// set the limitation of the cgroup on the specific subsystem
	Apply(path string, pid int) error	// add the process to the specific cgroup(specified by path)	
	Remove(path string) error // remove the cgroup
}

var(
	SubsystemIns = []Subsystem{
		&CpusetSubsystem{},
		&MemorySubsystem{},
		&CpuSubsystem{},
	}
)