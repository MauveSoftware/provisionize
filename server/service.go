package server

import (
	"context"

	"github.com/MauveSoftware/provisionize/api/proto"
)

// ProvisionService is a service interface for services participating in a provisioning
type ProvisionService interface {
	// Provision performs a step required to provision a virtual machine
	Provision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool

	// Deprovision performs a step required to deprovision a virtual machine
	Deprovision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool
}
