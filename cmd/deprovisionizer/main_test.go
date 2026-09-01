package main

import (
	"testing"

	"github.com/MauveSoftware/provisionize/pkg/api/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRequestFromParameters(t *testing.T) {
	*id = "42"
	*vmName = "test-vm"
	*clusterName = "cluster1"
	*fqdn = "test-vm.example.com"

	req := requestFromParameters()

	_, err := uuid.Parse(req.RequestId)
	assert.NoError(t, err, "request ID should be a valid UUID")

	assert.Equal(t, &proto.VirtualMachine{
		ClusterName: "cluster1",
		Id:          "42",
		Fqdn:        "test-vm.example.com",
		Name:        "test-vm",
	}, req.VirtualMachine)
}
