package subsystems

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

func FindCgroupMountPoint(subsystem string) string{
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil{
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan(){
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		for _, opt := range strings.Split(fields[len(fields) - 1], ","){
			if opt == subsystem{
				return fields[4]
			}
		}
	}
	if err := scanner.Err(); err != nil{
		return ""
	}
	return ""
}

func GetCgroupPath(subsystem string, cgroupPath string, autoCreate bool)(string, error){
	cgroupRoot := FindCgroupMountPoint(subsystem)
	if _, err := os.Stat(path.Join(cgroupRoot, cgroupPath)); err == nil || (autoCreate && os.IsNotExist(err)){
		if os.IsNotExist(err){
			if err := os.Mkdir(path.Join(cgroupRoot, cgroupPath), 0755); err != nil{
				return "", fmt.Errorf("error create cgroup %v", err)
			}
		}
		return path.Join(cgroupRoot, cgroupPath), nil
	}else{
		return "", fmt.Errorf("cgroup path error %v", err)
	}
}
/**  
MountInfo example:
	34 24 8:16 / / rw,relatime - ext4 /dev/sdb rw,discard,errors=remount-ro,data=ordered
	35 34 0:17 / /mnt/wsl rw,relatime shared:1 - tmpfs tmpfs rw
	36 34 0:18 /init /init ro,relatime - 9p tools ro,dirsync,aname=tools;fmask=022,loose,access=client,trans=fd,rfd=6,wfd=6
	37 34 0:6 / /dev rw,nosuid,relatime - devtmpfs none rw,size=4073128k,nr_inodes=1018282,mode=755
	38 34 0:16 / /sys rw,nosuid,nodev,noexec,noatime - sysfs sysfs rw
	39 34 0:21 / /proc rw,nosuid,nodev,noexec,noatime - proc proc rw
	40 37 0:22 / /dev/pts rw,nosuid,noexec,noatime - devpts devpts rw,gid=5,mode=620,ptmxmode=000
	41 34 0:23 / /run rw,nosuid,noexec,noatime - tmpfs none rw,mode=755
	42 41 0:24 / /run/lock rw,nosuid,nodev,noexec,noatime - tmpfs none rw
	43 41 0:25 / /run/shm rw,nosuid,nodev,noatime - tmpfs none rw
	44 41 0:26 / /run/user rw,nosuid,nodev,noexec,noatime - tmpfs none rw,mode=755
	45 39 0:19 / /proc/sys/fs/binfmt_misc rw,relatime - binfmt_misc binfmt_misc rw
	46 38 0:27 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755
	47 34 0:28 / /usr/lib/wsl/drivers ro,nosuid,nodev,noatime - 9p drivers ro,dirsync,aname=drivers;fmask=222;dmask=222,mmap,access=client,msize=65536,trans=fd,rfd=4,wfd=4
	48 34 0:29 / /usr/lib/wsl/lib ro,nosuid,nodev,noatime - 9p lib ro,dirsync,aname=lib;fmask=222;dmask=222,mmap,access=client,msize=65536,trans=fd,rfd=4,wfd=4
	49 46 0:30 / /sys/fs/cgroup/unified rw,nosuid,nodev,noexec,relatime - cgroup2 cgroup2 rw,nsdelegate
	50 46 0:31 / /sys/fs/cgroup/cpuset rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,cpuset
	51 46 0:32 / /sys/fs/cgroup/cpu rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,cpu
	52 46 0:33 / /sys/fs/cgroup/cpuacct rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,cpuacct
	53 46 0:34 / /sys/fs/cgroup/blkio rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,blkio
	54 46 0:35 / /sys/fs/cgroup/memory rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,memory
	55 46 0:36 / /sys/fs/cgroup/devices rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,devices
	56 46 0:37 / /sys/fs/cgroup/freezer rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,freezer
	57 46 0:38 / /sys/fs/cgroup/net_cls rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,net_cls
	58 46 0:39 / /sys/fs/cgroup/perf_event rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,perf_event
	59 46 0:40 / /sys/fs/cgroup/net_prio rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,net_prio
	60 46 0:41 / /sys/fs/cgroup/hugetlb rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,hugetlb
	61 46 0:42 / /sys/fs/cgroup/pids rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,pids
	62 46 0:43 / /sys/fs/cgroup/rdma rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,rdma
**/