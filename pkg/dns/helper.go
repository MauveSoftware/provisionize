package dns

import (
	"fmt"
	"net"
	"strings"
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
	var str strings.Builder

	for i := 3; i >= 0; i-- {
		str.WriteString(fmt.Sprintf("%d.", ip[i]))
	}

	return str.String() + "in-addr.arpa"
}

// IPv6ReverseDomain determines the reverse DNS domain for an IPv6 address
func IPv6ReverseDomain(ip net.IP) string {
	var str strings.Builder

	for i := 15; i >= 0; i-- {
		val := int(ip[i])
		p := 16
		for range 2 {
			str.WriteString(fmt.Sprintf("%x.", val%p))
			val /= p
			p *= 16
		}
	}

	return str.String() + "ip6.arpa"
}
