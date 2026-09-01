package proxmox

import (
	"testing"

	"github.com/MauveSoftware/provisionize/pkg/api/proto"
	api "github.com/Telmate/proxmox-api-go/proxmox"
	"github.com/stretchr/testify/assert"
)

func TestNetworkConfig(t *testing.T) {
	tests := []struct {
		name     string
		vm       *proto.VirtualMachine
		expected api.CloudInitNetworkInterfaces
	}{
		{
			name: "ipv4 and ipv6",
			vm: &proto.VirtualMachine{
				Ipv4: &proto.IPConfig{Address: "10.2.3.4", PrefixLength: 24},
				Ipv6: &proto.IPConfig{Address: "2001:678:1e0:f00::1", PrefixLength: 64},
			},
			expected: api.CloudInitNetworkInterfaces{
				api.QemuNetworkInterfaceID0: api.CloudInitNetworkConfig{
					IPv4: &api.CloudInitIPv4Config{Address: new(api.IPv4CIDR("10.2.3.4/24"))},
					IPv6: &api.CloudInitIPv6Config{Address: new(api.IPv6CIDR("2001:678:1e0:f00::1/64"))},
				},
			},
		},
		{
			name: "ipv4 only",
			vm: &proto.VirtualMachine{
				Ipv4: &proto.IPConfig{Address: "10.2.3.4", PrefixLength: 32},
			},
			expected: api.CloudInitNetworkInterfaces{
				api.QemuNetworkInterfaceID0: api.CloudInitNetworkConfig{
					IPv4: &api.CloudInitIPv4Config{Address: new(api.IPv4CIDR("10.2.3.4/32"))},
				},
			},
		},
		{
			name: "neither set",
			vm:   &proto.VirtualMachine{},
			expected: api.CloudInitNetworkInterfaces{
				api.QemuNetworkInterfaceID0: api.CloudInitNetworkConfig{},
			},
		},
	}

	t.Parallel()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := &ProxmoxService{}
			cfg := s.networkConfig(test.vm)

			assert.Equal(t, test.expected, cfg, "network config for %s did not match expectation", test.name)
		})
	}
}

func TestCreateVM_InvalidID(t *testing.T) {
	s := &ProxmoxService{}
	ch := make(chan *proto.StatusUpdate)

	ref, err := s.createVM(t.Context(), &proto.VirtualMachine{Id: "not-a-number"}, ch)

	assert.Nil(t, ref, "no VM ref should be returned when the ID is invalid")
	assert.EqualError(t, err, "ID has to be numeric")
}
