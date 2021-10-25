package network

import (
	"MyDocker/util"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
)

const ipamDefaultAllocatorPath = "/var/run/mydocker/network/ipam/subnet.json"

// IPAM means ip address management
type IPAM struct {
	SubnetAllocatorPath string
	Subnets             *map[string]*util.Bitmap // key是网络，value是分配的位图数组
}

var ipamDefaultAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

func (ipam *IPAM) load() error {
	// 不存在则不需要加载
	if !util.PathExists(ipam.SubnetAllocatorPath) {
		return nil
	}

	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	if err != nil {
		return fmt.Errorf("load ipam open %s failed: %v", ipam.SubnetAllocatorPath, err)
	}
	defer subnetConfigFile.Close()

	content, err := ioutil.ReadAll(subnetConfigFile)
	if err != nil {
		return fmt.Errorf("load ipam read %s failed: %v", ipam.SubnetAllocatorPath, err)
	}
	err = json.Unmarshal(content, ipam.Subnets)
	if err != nil {
		return fmt.Errorf("load ipam unmarshal failed: %v", err)
	}
	return nil
}

func (ipam *IPAM) dump() error {
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if !util.PathExists(ipamConfigFileDir) {
		if err := os.MkdirAll(ipamConfigFileDir, 0777); err != nil {
			return fmt.Errorf("dump ipam failed: %v", err)
		}
	}

	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	defer subnetConfigFile.Close()
	if err != nil {
		return fmt.Errorf("dump ipam open %s failed: %v", ipam.SubnetAllocatorPath, err)
	}

	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return fmt.Errorf("dump ipam marshal failed: %v", err)
	}
	_, err = subnetConfigFile.Write(ipamConfigJson)
	if err != nil {
		return fmt.Errorf("dump ipam write %s failed: %v", ipam.SubnetAllocatorPath, err)
	}
	return nil
}

// allocate an available addresss from network segment
func (ipam *IPAM) Allocate(subnet *net.IPNet) (net.IP, error) {
	ipam.Subnets = &map[string]*util.Bitmap{}
	err := ipam.load()
	if err != nil {
		return nil, fmt.Errorf("allocate failed: %v", err)
	}
	one, bits := subnet.Mask.Size()

	// 如果该网段未被分配,则初始化该网段
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		(*ipam.Subnets)[subnet.String()] = util.NewBitmap(1 << uint8(bits-one))
	}

	index, ok := (*ipam.Subnets)[subnet.String()].GetAvailableAndSet()
	if !ok {
		return nil, fmt.Errorf("allocate failed: no available ip in network segment")
	}
	ip := subnet.IP
	for i := 3; i >= 0; i-- {
		[]byte(ip)[3-i] += uint8(index >> (i * 8))
	}
	// ip从1开始分配，所以需要加1
	ip[3]++
	if err := ipam.dump(); err != nil {
		return nil, err
	}
	return ip, nil
}

func (ipam *IPAM) Release(subnet *net.IPNet, ipAddr *net.IP) error {
	ipam.Subnets = &map[string]*util.Bitmap{}
	err := ipam.load()
	if err != nil {
		return fmt.Errorf("allocate failed: %v", err)
	}
	releaseIP := ipAddr.To4()
	releaseIP[3]--
	index := 0
	for i := 3; i >= 0; i-- {
		index += int((releaseIP[i] - subnet.IP[i]) << ((3 - i) * 8))
	}
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		return fmt.Errorf("released ip not in ipam configfile")
	}
	println(index)
	(*ipam.Subnets)[subnet.String()].Remove(uint64(index))

	if err := ipam.dump(); err != nil {
		return err
	}
	return nil
}
