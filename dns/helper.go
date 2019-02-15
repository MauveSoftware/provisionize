package dns

import (
	"fmt"
	"net"
)

// ReverseDomain determines the reverse DNS domain for an IP address
func ReverseDomain(ip net.IP) string {
	if ip.To4() != nil {
		return IPv4ReverseDomain(ip.To4())
	}

	return IPv6ReverseDomain(ip.To16())
}

// IPv4ReverseDomain determines the reverse DNS domain for an IPv4 address
func IPv4ReverseDomain(ip net.IP) string {
	str := ""

	for i := 3; i >= 0; i-- {
		str += fmt.Sprintf("%d.", ip[i])
	}

	return str + "in-addr.arpa"
}

// IPv6ReverseDomain determines the reverse DNS domain for an IPv6 address
func IPv6ReverseDomain(ip net.IP) string {
	str := ""

	for i := 15; i >= 0; i-- {
		val := int(ip[i])
		p := 16
		for j := 0; j < 2; j++ {
			str += fmt.Sprintf("%x.", val%p)
			val /= p
			p *= 16
		}
	}

	return str + "ip6.arpa"
}
