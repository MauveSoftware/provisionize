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
	"golang.org/x/crypto/ssh"
)

const serviceName = "PVE"

// ProxmoxService is the service responsible for creating the virtual machine
type ProxmoxService struct {
	cl              *api.Client
	waitTimeout     time.Duration
	pollingInterval time.Duration
	user            string
	pass            string
	nodeIP          string
}

// NewService creates a new instance of ProxmoxService
func NewService(url, user, pass, nodeIP string) (*ProxmoxService, error) {
	timeout := 300 * time.Second

	tlsConf := &tls.Config{InsecureSkipVerify: true} // #nosec G402 -- Proxmox instances are typically deployed with self-signed certificates
	cl, err := api.NewClient(url, nil, "", tlsConf, "", int(timeout.Seconds()), false)
	if err != nil {
		return nil, fmt.Errorf("could not connect: %w", err)
	}

	s := &ProxmoxService{
		cl:              cl,
		waitTimeout:     timeout,
		pollingInterval: 10 * time.Second,
		user:            user,
		pass:            pass,
		nodeIP:          nodeIP,
	}

	return s, nil
}

// Provision creates the virtual machine
func (s *ProxmoxService) Provision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	ctx, span := trace.StartSpan(ctx, "ProxmoxService.Provision")
	defer span.End()

	err := s.login(ctx)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	ref, err := s.createVM(ctx, vm, ch)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: "Waiting for VM initialization to complete"}
	return s.waitForVMStatus(ctx, ref, "stopped", ch) &&
		s.initNetworkConfig(int(ref.VmId()), ch) &&
		s.startVM(ctx, ref, ch) &&
		s.waitForVMStatus(ctx, ref, "running", ch)
}

// Deprovision deletes the virtual machine
func (s *ProxmoxService) Deprovision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	ctx, span := trace.StartSpan(ctx, "ProxmoxService.Deprovision")
	defer span.End()

	err := s.login(ctx)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	ref, err := s.cl.GetVmRefByName(ctx, api.GuestName(vm.Name))
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	if ref == nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: fmt.Sprintf("VM %s does not exist: skipping", vm.Name)}
		return true
	}

	status, err := s.getVMStatus(ctx, ref)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	if status != "stopped" {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: fmt.Sprintf("VM is not down. Current status: %s", status)}
		return false
	}

	return s.deleteVM(ctx, ref, ch)
}

func (s *ProxmoxService) login(ctx context.Context) error {
	u := fmt.Sprintf("%s@pam", s.user)
	err := s.cl.Login(ctx, u, s.pass, "")
	if err != nil {
		return fmt.Errorf("could not authenticate: %w", err)
	}

	return nil
}

func (s *ProxmoxService) createVM(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) (*api.VmRef, error) {
	id, err := strconv.Atoi(vm.Id)
	if err != nil {
		return nil, fmt.Errorf("ID has to be numeric")
	}
	if id < api.GuestIdMinimum || id > api.GuestIdMaximum {
		return nil, fmt.Errorf("ID must be between %d and %d", api.GuestIdMinimum, api.GuestIdMaximum)
	}

	if vm.CpuCores < 1 || vm.CpuCores > 128 {
		return nil, fmt.Errorf("CPU cores must be between 1 and 128")
	}

	ch <- &proto.StatusUpdate{
		ServiceName: serviceName,
		Message:     "Creating VM by cloning template",
	}

	templateRef, err := s.cl.GetVmRefByName(ctx, api.GuestName(vm.Template))
	if err != nil {
		return nil, fmt.Errorf("could not get template: %w", err)
	}

	guestID := api.GuestID(id) // #nosec G115 -- id is range-checked above
	name := api.GuestName(vm.Name)

	config := &api.ConfigQemu{
		Name: &name,
		CPU: &api.QemuCPU{
			Sockets: new(api.QemuCpuSockets(1)),
			Cores:   new(api.QemuCpuCores(vm.CpuCores)), // #nosec G115 -- vm.CpuCores is range-checked above
		},
		Memory: &api.QemuMemory{
			CapacityMiB: new(api.QemuMemoryCapacity(vm.MemoryMb)),
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
	}

	ref, err := templateRef.CloneQemu(ctx, api.CloneQemuTarget{
		Full: &api.CloneQemuFull{
			Node: templateRef.Node(),
			ID:   &guestID,
			Name: &name,
		},
	}, s.cl)
	if err != nil {
		return nil, err
	}

	ch <- &proto.StatusUpdate{
		ServiceName: serviceName,
		Message:     "Updating VM configuration",
	}
	if err := s.cl.New().QemuGuest.Update(ctx, *ref, false, false, *config); err != nil {
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
			Address: new(api.IPv4CIDR(addr)),
		}
	}

	if vm.Ipv6 != nil {
		addr := fmt.Sprintf("%s/%d", vm.Ipv6.Address, vm.Ipv6.PrefixLength)
		netCfg.IPv6 = &api.CloudInitIPv6Config{
			Address: new(api.IPv6CIDR(addr)),
		}
	}

	cfg[api.QemuNetworkInterfaceID0] = netCfg

	return cfg
}

func (s *ProxmoxService) initNetworkConfig(id int, ch chan<- *proto.StatusUpdate) bool {
	ch <- &proto.StatusUpdate{
		ServiceName: serviceName,
		Message:     "Prepare network config before first start",
	}

	config := &ssh.ClientConfig{
		User: s.user,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // #nosec G106 -- connects to a fixed, operator-configured Proxmox node; host key is not currently pinned
	}

	addr := fmt.Sprintf("%s:22", s.nodeIP)
	sshCl, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		err = fmt.Errorf("could not connect to node: %w", err)
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	sess, err := sshCl.NewSession()
	if err != nil {
		err = fmt.Errorf("could not create SSH session: %w", err)
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	cmd := fmt.Sprintf("/var/lib/vz/snippets/network-hookscript.pl %d init", id)
	err = sess.Run(cmd)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	return true
}

func (s *ProxmoxService) waitForVMStatus(ctx context.Context, ref *api.VmRef, desiredStatus string, ch chan<- *proto.StatusUpdate) bool {
	currentStatus := ""

	for {
		select {
		case <-time.After(s.waitTimeout):
			ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: "Operation timed out"}
			return false

		case <-time.After(s.pollingInterval):
			status, err := s.getVMStatus(ctx, ref)
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

func (s *ProxmoxService) getVMStatus(ctx context.Context, ref *api.VmRef) (string, error) {
	st, err := ref.GetRawGuestStatus(ctx, s.cl)
	if err != nil {
		return "", err
	}

	return st.GetState().String(), nil
}

func (s *ProxmoxService) startVM(ctx context.Context, ref *api.VmRef, ch chan<- *proto.StatusUpdate) bool {
	err := s.cl.New().Guest.Start(ctx, *ref)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: "VM started"}
	return true
}

func (s *ProxmoxService) deleteVM(ctx context.Context, ref *api.VmRef, ch chan<- *proto.StatusUpdate) bool {
	existed, err := s.cl.New().Guest.Delete(ctx, *ref)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	msg := "VM deletion initiated"
	if !existed {
		msg = "VM did not exist"
	}
	ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: msg}
	return true
}
