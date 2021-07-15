package ovirt

import (
	"github.com/MauveSoftware/provisionize/pkg/api/proto"
)

// ConfigService encapsulates the configuration which depents on the type of VM
type ConfigService interface {
	// OvirtTemplateNameForVM returns the ovirt template name for an VM
	OvirtTemplateNameForVM(vm *proto.VirtualMachine) string

	// BootDiskName returns the name of the boot diks
	BootDiskName(vm *proto.VirtualMachine) string
}
