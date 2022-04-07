package addr

import (
	"fmt"
	"net"
)

func IsPrivateIP(addr string) bool {
	ip := net.ParseIP(addr)
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsPrivate()
}

// IsLocal tells us whether an ip is local
func IsLocal(addr string) bool {
	// extract the host
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		addr = host
	}

	// check if its localhost
	if addr == "localhost" {
		return true
	}

	// check against all local ips
	for _, ip := range LocalIPs() {
		if addr == ip {
			return true
		}
	}

	return false
}

// Extract returns a real ip
func Extract(addr string) (string, error) {
	// if addr specified then its returned
	if len(addr) > 0 && (addr != "0.0.0.0" && addr != "[::]" && addr != "::") {
		return addr, nil
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get interfaces err: %w", err)
	}

	var addresses []net.Addr
	for _, iface := range interfaces {
		interfaceAddresses, err := iface.Addrs()
		// ignore error, interface can disappear from system
		if err != nil {
			continue
		}
		addresses = append(addresses, interfaceAddresses...)
	}

	var localAddr, publicIP string
	for _, rawAddr := range addresses {
		var ip net.IP
		switch addr := rawAddr.(type) {
		case *net.IPAddr:
			ip = addr.IP
		case *net.IPNet:
			ip = addr.IP
		default:
			continue
		}

		if ip := ip.String(); !IsPrivateIP(ip) {
			publicIP = ip
			continue
		} else {
			localAddr = ip
			break
		}
	}

	// return private ip
	if len(localAddr) > 0 {
		a := net.ParseIP(localAddr)
		if a == nil {
			return "", fmt.Errorf("ip addr %s is invalid", localAddr)
		}
		return a.String(), nil
	}

	// return public or virtual ip
	if len(publicIP) > 0 {
		a := net.ParseIP(publicIP)
		if a == nil {
			return "", fmt.Errorf("ip addr %s is invalid", publicIP)
		}
		return a.String(), nil
	}
	return "", fmt.Errorf("no ip address found, and explicit ip not provided")
}

// LocalIPs returns all known ips
func LocalIPs() (out []string) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, i := range interfaces {
		addresses, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addresses {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			out = append(out, ip.String())
		}
	}
	return
}
