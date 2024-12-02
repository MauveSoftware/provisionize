package proxmox

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"
	"time"

	"github.com/MauveSoftware/provisionize/pkg/api/proto"
	api "github.com/Telmate/proxmox-api-go/proxmox"
	"go.opencensus.io/trace"
)

const serviceName = "PVE"

// ProxmoxService is the service responsible for creating the virtual machine
type ProxmoxService struct {
	cl              *api.Client
	waitTimeout     time.Duration
	pollingInterval time.Duration
}

// NewService creates a new instance of ProxmoxService
func NewService(url, user, pass string) (*ProxmoxService, error) {
	timeout := 300 * time.Second

	tlsConf := &tls.Config{InsecureSkipVerify: true}
	cl, err := api.NewClient(url, nil, "", tlsConf, "", int(timeout.Seconds()))
	if err != nil {
		return nil, fmt.Errorf("could not connect: %w", err)
	}

	err = cl.Login(user, pass, "")
	if err != nil {
		return nil, fmt.Errorf("could not authenticate: %w", err)
	}

	s := &ProxmoxService{
		cl:              cl,
		waitTimeout:     timeout,
		pollingInterval: 10 * time.Second,
	}

	return s, nil
}

// Provision creates the virtual machine
func (s *ProxmoxService) Provision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	ctx, span := trace.StartSpan(ctx, "ProxmoxService.Provision")
	defer span.End()

	ref, err := s.createVM(vm, ch)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: "Waiting for VM initialization to complete"}
	return s.waitForVMStatus(ref, "stopped", ch) &&
		s.startVM(ref, ch) &&
		s.waitForVMStatus(ref, "running", ch)
}

// Deprovision deletes the virtual machine
func (s *ProxmoxService) Deprovision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	ctx, span := trace.StartSpan(ctx, "ProxmoxService.Deprovision")
	defer span.End()

	ref, err := s.cl.GetVmRefByName(vm.Name)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	if ref == nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: fmt.Sprintf("VM %s does not exist: skipping", vm.Name)}
		return true
	}

	status, err := s.getVMStatus(ref)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	if status != "stopped" {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: fmt.Sprintf("VM is not down. Current status: %s", status)}
		return false
	}

	return s.deleteVM(ref, ch)
}

func (s *ProxmoxService) createVM(vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) (*api.VmRef, error) {
	id, err := strconv.Atoi(vm.Id)
	if err != nil {
		return nil, fmt.Errorf("ID has to be numeric")
	}

	ch <- &proto.StatusUpdate{
		ServiceName: serviceName,
		Message:     "Creating VM by cloning template",
	}

	templateRef, err := s.cl.GetVmRefByName(vm.Template)
	if err != nil {
		return nil, fmt.Errorf("could not get template: %w", err)
	}

	ref := api.NewVmRef(id)

	config := &api.ConfigQemu{
		Name: vm.Name,
		CPU: &api.QemuCPU{
			Sockets: pointer(api.QemuCpuSockets(1)),
			Cores:   pointer(api.QemuCpuCores(vm.CpuCores)),
		},
		Memory: &api.QemuMemory{
			CapacityMiB: pointer(api.QemuMemoryCapacity(vm.MemoryMb)),
		},
		CloudInit: &api.CloudInit{
			NetworkInterfaces: s.networkConfig(vm),
			Custom: &api.CloudInitCustom{
				Network: &api.CloudInitSnippet{
					FilePath: api.CloudInitSnippetPath(fmt.Sprintf("snippets/ci-network-%d.yml", id)),
					Storage:  "local",
				},
			},
		},
		FullClone: pointer(1),
	}

	err = config.CloneVm(templateRef, ref, s.cl)
	if err != nil {
		return nil, err
	}

	ch <- &proto.StatusUpdate{
		ServiceName: serviceName,
		Message:     "Updating VM configuration",
	}
	_, err = config.Update(false, ref, s.cl)
	if err != nil {
		return nil, err
	}

	ch <- &proto.StatusUpdate{
		ServiceName: serviceName,
		Message:     "VM created successfully",
	}

	return ref, nil
}

func (s *ProxmoxService) networkConfig(vm *proto.VirtualMachine) api.CloudInitNetworkInterfaces {
	cfg := api.CloudInitNetworkInterfaces{}

	netCfg := api.CloudInitNetworkConfig{}
	if vm.Ipv4 != nil {
		addr := fmt.Sprintf("%s/%d", vm.Ipv4.Address, vm.Ipv4.PrefixLength)
		netCfg.IPv4 = &api.CloudInitIPv4Config{
			Address: pointer(api.IPv4CIDR(addr)),
		}
	}

	if vm.Ipv6 != nil {
		addr := fmt.Sprintf("%s/%d", vm.Ipv6.Address, vm.Ipv6.PrefixLength)
		netCfg.IPv6 = &api.CloudInitIPv6Config{
			Address: pointer(api.IPv6CIDR(addr)),
		}
	}

	cfg[api.QemuNetworkInterfaceID0] = netCfg

	return cfg
}

func (s *ProxmoxService) waitForVMStatus(ref *api.VmRef, desiredStatus string, ch chan<- *proto.StatusUpdate) bool {
	currentStatus := ""

	for {
		select {
		case <-time.After(s.waitTimeout):
			ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: "Operation timed out"}
			return false

		case <-time.After(s.pollingInterval):
			status, err := s.getVMStatus(ref)
			if err != nil {
				ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
				return false
			}

			if status != currentStatus {
				ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: fmt.Sprintf("New status: %s", status)}
				currentStatus = status
			}

			if status == desiredStatus {
				return true
			}
		}
	}
}

func (s *ProxmoxService) getVMStatus(ref *api.VmRef) (string, error) {
	st, err := s.cl.GetVmState(ref)
	if err != nil {
		return "", err
	}

	return st["status"].(string), nil
}

func (s *ProxmoxService) startVM(ref *api.VmRef, ch chan<- *proto.StatusUpdate) bool {
	exitStatus, err := s.cl.StartVm(ref)

	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error(), DebugMessage: exitStatus}
		return false
	}

	ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: "VM started", DebugMessage: exitStatus}
	return true
}

func (s *ProxmoxService) deleteVM(ref *api.VmRef, ch chan<- *proto.StatusUpdate) bool {
	existStatus, err := s.cl.DeleteVm(ref)

	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error(), DebugMessage: existStatus}
		return false
	}

	ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: "VM deletion initiated", DebugMessage: existStatus}
	return true
}
