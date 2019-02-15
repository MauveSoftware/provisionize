package server

import (
	"context"

	"github.com/MauveSoftware/provisionize/api/proto"
)

// ProvisionService is a service interface for services participating in a provisioning
type ProvisionService interface {
	PerformStep(ctx context.Context, vm *proto.VirtualMachine) *proto.ServiceResult
}
