package dns

import (
	"net"

	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReverseDomain(t *testing.T) {
	tests := []struct {
		name     string
		ip       net.IP
		expected string
	}{
		{
			name:     "ipv4",
			ip:       net.ParseIP("185.138.53.1"),
			expected: "1.53.138.185.in-addr.arpa",
		},
		{
			name:     "ipv6",
			ip:       net.ParseIP("2001:678:1e0:100::200:1"),
			expected: "1.0.0.0.0.0.2.0.0.0.0.0.0.0.0.0.0.0.1.0.0.e.1.0.8.7.6.0.1.0.0.2.ip6.arpa",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ReverseDomain(test.ip)
			assert.Equal(t, test.expected, result)
		})
	}
}
