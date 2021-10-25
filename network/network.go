package network

import (
	"MyDocker/container"
	"MyDocker/util"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

var (
	defaultNetworkPath = "/var/run/mydocker/network/network"
	drivers            = map[string]NetworkDriver{}
	networks           = map[string]*Network{}
)

type Network struct {
	Name    string
	IPRange *net.IPNet
	Driver  string
}

type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	PortMapping []string         `json:"portmapping"`
	Network     *Network
}

type NetworkDriver interface {
	Name() string                                         // 驱动名
	Create(subsnet string, name string) (*Network, error) // 创建网络
	Delete(network Network) error                         // 删除网络
	Connect(network *Network, endpoint *Endpoint) error   // 连接容器网络端点到网络
	Disconnect(network Network, endpoint *Endpoint) error // 从网络上移除容器网络端点
}

func CreateNetwork(driver string, subnet string, name string) error {
	_, cidr, err := net.ParseCIDR(subnet)
	if err != nil {
		return err
	}
	// 使用IPAM获取可用IP
	gatewayIP, err := ipamDefaultAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = gatewayIP

	// 调用指定的驱动创建网络
	network, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}
	return network.dump(defaultNetworkPath)
}

func (network *Network) dump(dumpPath string) error {
	if !util.PathExists(dumpPath) {
		if err := os.MkdirAll(dumpPath, 0666); err != nil {
			return fmt.Errorf("dump failed: %v", err)
		}
	}
	networkPath := path.Join(dumpPath, network.Name)
	networkFile, err := os.OpenFile(networkPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("dump failed: %v", err)
	}
	defer networkFile.Close()

	networkJSON, err := json.Marshal(network)
	if err != nil {
		return fmt.Errorf("dump failed: marshal failed: %v", err)
	}

	// write network in json format
	_, err = networkFile.Write(networkJSON)
	if err != nil {
		return fmt.Errorf("dump failed: %v", err)
	}
	return nil
}

func (network *Network) load(networkPath string) error {
	networkFile, err := os.Open(networkPath)
	if err != nil {
		return fmt.Errorf("load failed: %v", err)
	}
	defer networkFile.Close()

	networkJSON, err := ioutil.ReadAll(networkFile)
	if err != nil {
		return fmt.Errorf("load failed: %v", err)
	}

	err = json.Unmarshal(networkJSON, network)
	if err != nil {
		return fmt.Errorf("load failed: %v", err)
	}
	return nil
}

func (network *Network) remove(networkPath string) error {
	return os.Remove(path.Join(networkPath, network.Name))
}

// Init loads networky to  memory
func Init() error {
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	if !util.PathExists(defaultNetworkPath) {
		if err := os.MkdirAll(defaultNetworkPath, 0666); err != nil {
			return fmt.Errorf("init network failed: %v", err)
		}
	}

	filepath.Walk(defaultNetworkPath, func(networkPath string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		_, networkFileName := path.Split(networkPath)
		network := &Network{}

		if err := network.load(networkPath); err != nil {
			return fmt.Errorf("init network failed: %v", err)
		}

		// add it to the networks directory
		networks[networkFileName] = network
		return nil
	})
	return nil
}

// For command: network list
func ListNetwork() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "IPRange", "Driver"})
	for _, network := range networks {
		table.Append([]string{network.Name, network.IPRange.String(), network.Driver})
	}
	table.Render()
}

// For command: network delete
func DeleteNetwork(networkName string) error {
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no such network: %s", networkName)
	}

	// 释放网关的ip地址
	if err := ipamDefaultAllocator.Release(network.IPRange, &network.IPRange.IP); err != nil {
		return fmt.Errorf("romove network release ip failed: %v", err)
	}

	//调用网络驱动删除网络创建的设备与配置
	if err := drivers[network.Driver].Delete(*network); err != nil {
		return fmt.Errorf("remove network failed: %v", err)
	}

	return network.remove(defaultNetworkPath)
}

//
func Connect(networkName string, cinfo *container.ContainerInfo) error {
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no suck network: %s", networkName)
	}

	//为容器分配ip
	ip, err := ipamDefaultAllocator.Allocate(network.IPRange)
	if err != nil {
		return err
	}
	log.Infof("allocated ip is %v", ip)

	// 创建网络端点
	endpoint := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.ID, networkName),
		IPAddress:   ip,
		Network:     network,
		PortMapping: cinfo.PortMapping,
	}

	// 调用网络对应的网络驱动挂载，配置网络端点
	if err = drivers[network.Driver].Connect(network, endpoint); err != nil {
		return err
	}

	// 到容器的Namespace中配置容器网络、设备IP地址和路由信息
	if err = configEndpointIPAddressAndRoute(endpoint, cinfo); err != nil {
		return err
	}

	// 配置端口映射信息
	return configPortMapping(endpoint, cinfo)
}

//配置容器网络端点的地址和路由
func configEndpointIPAddressAndRoute(endpoint *Endpoint, cinfo *container.ContainerInfo) error {
	peerLink, err := netlink.LinkByName(endpoint.Device.PeerName)
	if err != nil {
		return fmt.Errorf("configEndpoint failed: %v", err)
	}

	// 将容器的网络端口加入到容器网络空间中，同时使函数下面的操作都在此网络空间中进行
	// 执行完函数后，恢复为默认的网络空间
	defer enterContainerNetns(&peerLink, cinfo)()

	//获取容器的IP地址以及网段
	interfaceIP := *endpoint.Network.IPRange
	interfaceIP.IP = endpoint.IPAddress

	//调用setInterfaceIP设置容器内Veth端点的IP
	if err = setInterfaceIP(endpoint.Device.PeerName, &interfaceIP); err != nil {
		return fmt.Errorf("configEndpoint failed: %v", err)
	}

	// 启动容器内的Veth端点
	if err = setInterfaceUP(endpoint.Device.PeerName); err != nil {
		return fmt.Errorf("configEndpoint failed: %v", err)
	}

	// 本地回环io默认是关闭的，需要启动它
	if err = setInterfaceUP("lo"); err != nil {
		return fmt.Errorf("configEndpoint failed: %v", err)
	}

	//设置容器内的外部请求都通过Veth端点进行访问
	_, cidr, _ := net.ParseCIDR("::/0")
	fmt.Println(cidr.IP.String())
	fmt.Println(endpoint.Network.IPRange.IP.String())

	// 相当于 route add -net 0.0.0.0/0 gw {bridge网桥地址} dev {容器内Veth端点设备}
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        endpoint.Network.IPRange.IP,
		Dst:       cidr,
	}

	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return fmt.Errorf("configEndpoint failed: %v", err)
	}
	return nil
}

// 配置宿主机到容器的端口映射
func configPortMapping(endpoint *Endpoint, cinfo *container.ContainerInfo) error {
	for _, pm := range endpoint.PortMapping {
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			return fmt.Errorf("invalid port mapping format")
		}
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s %s", portMapping[0], endpoint.IPAddress.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("iptables error: %v", output)
		}
	}
	return nil
}

// 将容器的网络端点加入到容器的网络空间中去
// 并锁定当前程序执行的线程，使当前线程加入到容器的网络空间
func enterContainerNetns(enLink *netlink.Link, cinfo *container.ContainerInfo) func() {
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.PID), os.O_RDONLY, 0)
	if err != nil {
		log.Errorf("get container ns failed: %v", err)
	}

	nsFD := f.Fd()

	//锁定当前程序所执行线程，如果不锁定操作系统线程，goroutine可能会被调度到别的线程上去，此时无法保证其所在的网络空间
	runtime.LockOSThread()

	//修改网络端点Veth的另外一端，将其移动到容器的 Net Namespace 中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		log.Errorf("set link netns failed: %v", err)
	}

	// 获取当前网络的namespace
	origins, err := netns.Get()
	if err != nil {
		log.Errorf("get current netns failed: %v", err)
	}

	//将当前进程加入到容器的Net Namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		log.Errorf("set netns failed: %v", err)
	}

	// 返回之前的Net Namespace
	return func() {
		// 使用上面的origins进行恢复
		netns.Set(origins)
		origins.Close()
		// 取消锁定
		runtime.UnlockOSThread()
		// 关闭Namespace文件
		f.Close()
	}
}
