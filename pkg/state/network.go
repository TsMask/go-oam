package state

import (
	goNet "net"
	"strings"
)

// NetIface 网络接口信息
type NetIface struct {
	Name string   `json:"name"` // 接口名称
	IPv4 []string `json:"ipv4"` // IPv4 地址列表
	IPv6 []string `json:"ipv6"` // IPv6 地址列表
}

// shouldSkipIface 判断是否跳过该接口（回环、链路本地、虚拟接口）。
func shouldSkipIface(iface goNet.Interface) bool {
	if iface.Flags&goNet.FlagLoopback != 0 {
		return true
	}
	name := iface.Name
	return strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "docker")
}

// parseIfaceIPs 从网卡地址列表中提取并分类 IPv4/IPv6，过滤链路本地地址。
func parseIfaceIPs(addrs []goNet.Addr) (ipv4 []string, ipv6 []string) {
	for _, addr := range addrs {
		ip, _, err := goNet.ParseCIDR(addr.String())
		if err != nil {
			continue
		}
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}
		ipStr := ip.String()
		if ip.To4() != nil {
			ipv4 = append(ipv4, ipStr)
		} else {
			ipv6 = append(ipv6, ipStr)
		}
	}
	return
}

// GetSystemNetwork 获取各网卡的 IPv4/IPv6 地址。
// 跳过回环、链路本地和虚拟接口（veth、docker）。
func SystemNetwork() []NetIface {
	ifaces, err := goNet.Interfaces()
	if err != nil {
		return nil
	}
	result := make([]NetIface, 0, len(ifaces))
	for _, iface := range ifaces {
		if shouldSkipIface(iface) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		ipv4, ipv6 := parseIfaceIPs(addrs)
		if len(ipv4) > 0 || len(ipv6) > 0 {
			result = append(result, NetIface{
				Name: iface.Name,
				IPv4: ipv4,
				IPv6: ipv6,
			})
		}
	}
	return result
}

// NetworkDevices 获取网卡设备信息，返回前端树形结构。
// 每个节点包含 id（索引）、label（接口名）、mac（物理地址）及 addrs（IP 列表）。
func NetworkDevices() []map[string]any {
	arr := make([]map[string]any, 0)
	ifaces, err := goNet.Interfaces()
	if err != nil {
		return arr
	}
	for _, iface := range ifaces {
		if iface.Flags&goNet.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		ipv4, ipv6 := parseIfaceIPs(addrs)
		addrArr := append(ipv4, ipv6...)
		if len(addrArr) == 0 {
			continue
		}
		arr = append(arr, map[string]any{
			"id":    iface.Index,
			"label": iface.Name,
			"mac":   iface.HardwareAddr.String(),
			"addrs": addrArr,
		})
	}
	return arr
}
