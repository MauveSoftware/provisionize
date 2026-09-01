package main

import (
	"net"
	"testing"

	"github.com/MauveSoftware/provisionize/pkg/api/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRequestFromParameters(t *testing.T) {
	*id = "42"
	*vmName = "test-vm"
	*clusterName = "cluster1"
	*templateName = "ubuntu-18-04"
	*fqdn = "test-vm.example.com"
	*ipv4 = net.ParseIP("10.2.3.4")
	*ipv4Gateway = net.ParseIP("10.2.3.1")
	*ipv4PfxLen = 24
	*ipv6 = net.ParseIP("2001:678:1e0:f00::1")
	*ipv6Gateway = net.ParseIP("2001:678:1e0:f00::fffe")
	*ipv6PfxLen = 64
	*cores = 2
	*memory = 2048

	req := requestFromParameters()

	_, err := uuid.Parse(req.RequestId)
	assert.NoError(t, err, "request ID should be a valid UUID")

	assert.Equal(t, &proto.VirtualMachine{
		ClusterName: "cluster1",
		CpuCores:    2,
		Id:          "42",
		Fqdn:        "test-vm.example.com",
		Ipv4: &proto.IPConfig{
			Address:      "10.2.3.4",
			PrefixLength: 24,
			Gateway:      "10.2.3.1",
		},
		Ipv6: &proto.IPConfig{
			Address:      "2001:678:1e0:f00::1",
			PrefixLength: 64,
			Gateway:      "2001:678:1e0:f00::fffe",
		},
		MemoryMb: 2048,
		Name:     "test-vm",
		Template: "ubuntu-18-04",
	}, req.VirtualMachine)
}
