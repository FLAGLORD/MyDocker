package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
)

type BridgeNetworkDriver struct{}

func (driver *BridgeNetworkDriver) Name() string {
	return "bridge"
}

func (driver *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	//  netlink.ParseIPNet是对net.ParseCIDR的封装
	ipRange, err := netlink.ParseIPNet(subnet)
	if err != nil {
		return nil, fmt.Errorf("network create failed: %v", err)
	}
	network := &Network{
		Name:    name,
		IPRange: ipRange,
		Driver:  driver.Name(),
	}

	if err := driver.initBridge(network); err != nil {
		return nil, fmt.Errorf("network create failed: %v", err)
	}
	return network, nil
}

func (driver *BridgeNetworkDriver) Delete(network Network) error {
	bridgeName := network.Name
	// find bridge
	bridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	return netlink.LinkDel(bridge)
}

func (driver *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	bridgeName := network.Name
	// 通过设备名获取bridge对象
	bridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	// 创建Veth
	linkAttrs := netlink.NewLinkAttrs()
	// 取endpoint ID的前5位
	linkAttrs.Name = endpoint.ID[:5]
	// 设置veth接口的master属性，设置此veth的一端挂载到对应的bridge上
	linkAttrs.MasterIndex = bridge.Attrs().Index

	//创建Veth对象，通过PeerName配置Veth另一端的接口名
	endpoint.Device = netlink.Veth{
		LinkAttrs: linkAttrs,
		PeerName:  "cif-" + endpoint.ID[:5],
	}

	// 调用LinkAdd创建Veth
	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("bridgeConnect faild： %v", err)
	}

	// 启动Veth
	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("bridgeConnect faild： %v", err)
	}
	return nil
}

func (driver *BridgeNetworkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	return nil
}

// initBridge inits dev bridge
func (driver *BridgeNetworkDriver) initBridge(n *Network) error {
	// 1. create bridge
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("initBridge failed: %v", err)
	}

	// 2. set ip and route for bridge
	if err := setInterfaceIP(bridgeName, n.IPRange); err != nil {
		return fmt.Errorf("initBridge failed: %v", err)
	}

	// 3. enables bridge
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("initBridge failed: %v", err)
	}

	// 4. set iptables SNAT rules
	if err := setIPTables(bridgeName, n.IPRange); err != nil {
		return fmt.Errorf("initBridge failed: %v", err)
	}
	return nil
}

// createBridgeInterface creates linux bridge
func createBridgeInterface(bridgeName string) error {
	// 检查是否存在同名的bridge
	_, err := net.InterfaceByName(bridgeName)
	// 已存在或者报告其他错误
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = bridgeName
	// 创建bridge对象
	bridge := &netlink.Bridge{
		LinkAttrs: linkAttrs,
	}
	// 创建虚拟网络设备，相当于 ip link add xxxx
	if err := netlink.LinkAdd(bridge); err != nil {
		return fmt.Errorf("bridgeCreate %s failed：%v", bridgeName, err)
	}

	return nil
}

// setInterfaceIP 设置网络接口的IP地址
func setInterfaceIP(name string, ipNet *net.IPNet) error {
	retries := 2
	var netInterface netlink.Link
	var err error
	for i := 0; i < retries; i++ {
		netInterface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("get interface failed: %v", err)
	}

	// netlink.AddrAdd相当于调用 ip addr addd xxx
	// 同时如果配置地址所在网段的信息，还会配置路由表 该网段的信息会转发到该此网络接口上
	addr := &netlink.Addr{
		IPNet:     ipNet,
		Peer:      ipNet,
		Label:     "",
		Flags:     0,
		Scope:     0,
		Broadcast: nil,
	}
	return netlink.AddrAdd(netInterface, addr)
}

// setInterfaceUP 启动网络设备
func setInterfaceUP(name string) error {
	netInterface, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("setInterfaceUP get interface failed: %v", err)
	}

	// netlink.LinkSetUp() 相当于调用 ip link set xxx up
	if err := netlink.LinkSetUp(netInterface); err != nil {
		return fmt.Errorf("setInterfaceUP enables interface failed: %v", err)
	}
	return nil
}

// setIPTables为bridge设置对应的 MASQUERADE 规则
func setIPTables(name string, subnet *net.IPNet) error {
	// iptables -t nat -A POSTROUTING -s <bridgeNetworkSegment> ! -o <bridgeName> -j MASQUERADE
	// -o means --out-interface, refer to https://linux.die.net/man/8/iptables
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), name)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("setIPtables failed: %v", err)
	}
	return nil
}
