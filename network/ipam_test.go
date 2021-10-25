package network

import (
	"net"
	"testing"
)

func TestAllocate(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.0.1/24")
	ip, err := ipamDefaultAllocator.Allocate(ipnet)
	if err != nil {
		t.Error(err)
	}
	t.Logf("alloc ip: %v", ip)
}

func TestRelease(t *testing.T) {
	ip, ipnet, _ := net.ParseCIDR("192.168.0.1/24")
	err := ipamDefaultAllocator.Release(ipnet, &ip)
	if err != nil {
		t.Error(err)
	}
}
 