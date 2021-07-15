package tower

import (
	"github.com/MauveSoftware/provisionize/pkg/api/proto"
)

// ConfigService encapsulates the configuration which depents on the type of VM
type ConfigService interface {
	// TowerTemplateIDsForVM return the job template ids to launch for the VM
	TowerTemplateIDsForVM(vm *proto.VirtualMachine) []uint
}
