package main

import (
	"math"
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

	req, err := requestFromParameters()
	assert.NoError(t, err, "requestFromParameters should not fail for valid flag values")

	_, err = uuid.Parse(req.RequestId)
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

func TestUint32FromFlag(t *testing.T) {
	tests := []struct {
		name        string
		value       uint
		expectError bool
	}{
		{name: "in range", value: 4096},
		{name: "exceeds uint32", value: math.MaxUint32 + 1, expectError: true},
	}

	t.Parallel()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v, err := uint32FromFlag("cores", test.value)

			if test.expectError {
				assert.Error(t, err, "expected an error for value %d", test.value)
				return
			}

			assert.NoError(t, err, "did not expect an error for value %d", test.value)
			assert.Equal(t, uint32(test.value), v)
		})
	}
}
