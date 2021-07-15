package ovirt

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"

	"github.com/MauveSoftware/provisionize/pkg/api/proto"
	ovirt "github.com/czerwonk/ovirt_api/api"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

const serviceName = "oVirt"

// OvirtService is the service responsible for creating the virtual machine
type OvirtService struct {
	template        string
	configService   ConfigService
	client          *ovirt.Client
	waitTimeout     time.Duration
	pollingInterval time.Duration
}

// NewService creates a new instance of OvirtService
func NewService(url, user, pass string, template string, configService ConfigService) (*OvirtService, error) {
	client, err := ovirt.NewClient(url, user, pass, ovirt.WithDebug(), ovirt.WithInsecure())
	if err != nil {
		return nil, errors.Wrap(err, "could not create new oVirt client")
	}

	svc := &OvirtService{
		client:          client,
		template:        template,
		waitTimeout:     2 * time.Minute,
		pollingInterval: 10 * time.Second,
		configService:   configService,
	}

	return svc, nil
}

// Provision creates the virtual machine
func (s *OvirtService) Provision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	ctx, span := trace.StartSpan(ctx, "OvirtService.Provision")
	defer span.End()

	b, err := s.createVM(vm, ch)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	v := &VM{}
	err = xml.Unmarshal(b, &v)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: "Waiting for VM initialization to complete"}
	return s.waitForVMStatus(v.ID, "down", ch) &&
		s.ensureBootDiskIsAttached(vm, v.ID, ch) &&
		s.startVM(v.ID, ch) &&
		s.waitForVMStatus(v.ID, "up", ch)
}

// Deprovision deletes the virtual machine
func (s *OvirtService) Deprovision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	v, err := s.getVMByName(vm.Name)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	if v == nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: fmt.Sprintf("VM %s does not exist: skipping", vm.Name)}
		return true
	}

	if v.Status != "down" {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: fmt.Sprintf("VM is not down. Current status: %s", v.Status)}
		return false
	}

	return s.deleteVM(v.ID, ch) && s.waitForVanish(v.ID, ch)
}

func (s *OvirtService) createVM(vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) ([]byte, error) {
	body, err := s.getVMCreateRequest(vm)
	if err != nil {
		return nil, err
	}

	ch <- &proto.StatusUpdate{
		ServiceName:  serviceName,
		DebugMessage: body.String(),
		Message:      "Start creating VM",
	}

	b, err := s.sendCreateRequestWithRetry(body)
	if err != nil {
		return nil, err
	}

	ch <- &proto.StatusUpdate{
		ServiceName:  serviceName,
		DebugMessage: string(b),
		Message:      "VM created successfully",
	}

	return b, nil
}

func (s *OvirtService) sendCreateRequestWithRetry(body io.Reader) ([]byte, error) {
	isRetry := false
	for {
		b, err := s.client.SendRequest("vms?clone=true", "POST", body)
		if err == nil {
			return b, nil
		}

		if err.Error() == "400 Bad Request" && !isRetry {
			// oVirt returns 400 if API is not ready, so we retry on this code one more time
			time.Sleep(100)
			isRetry = true
			continue
		}

		return nil, err
	}
}

func (s *OvirtService) getVMCreateRequest(vm *proto.VirtualMachine) (*bytes.Buffer, error) {
	funcs := template.FuncMap{
		"mb_to_byte": func(x uint32) uint64 {
			return uint64(x) * (1 << 20)
		},
		"ovirt_template_name": func() string {
			return s.configService.OvirtTemplateNameForVM(vm)
		},
	}
	tmpl, err := template.New("create-vm").Funcs(funcs).Parse(s.template)
	if err != nil {
		return nil, err
	}

	w := &bytes.Buffer{}
	err = tmpl.Execute(w, vm)
	return w, err
}

func (s *OvirtService) waitForVMStatus(id string, desiredStatus string, ch chan<- *proto.StatusUpdate) bool {
	currentStatus := ""

	for {
		select {
		case <-time.After(s.waitTimeout):
			ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: "Operation timed out"}
			return false

		case <-time.After(s.pollingInterval):
			vm, err := s.getVM(id)
			if err != nil {
				ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
				return false
			}

			if vm.Status != currentStatus {
				ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: fmt.Sprintf("New status: %s", vm.Status)}
				currentStatus = vm.Status
			}

			if vm.Status == desiredStatus {
				return true
			}
		}
	}
}

func (s *OvirtService) waitForVanish(id string, ch chan<- *proto.StatusUpdate) bool {
	for {
		select {
		case <-time.After(s.waitTimeout):
			ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: "Operation timed out"}
			return false

		case <-time.After(s.pollingInterval):
			vm, err := s.getVM(id)
			if err != nil && err.Error() != "404 Not Found" {
				ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
				return false
			}

			if vm == nil {
				ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: "VM deleted"}
				return true
			}
		}
	}
}

func (s *OvirtService) getVM(id string) (*VM, error) {
	var vm VM
	err := s.client.GetAndParse(fmt.Sprintf("vms/%s", id), &vm)
	if err != nil {
		return nil, err
	}

	return &vm, nil
}

func (s *OvirtService) getVMByName(name string) (*VM, error) {
	var vms VMs
	err := s.client.GetAndParse("vms", &vms)
	if err != nil {
		return nil, err
	}

	for _, vm := range vms.VMs {
		if vm.Name == name {
			return &vm, nil
		}
	}

	return nil, nil
}

func (s *OvirtService) ensureBootDiskIsAttached(vm *proto.VirtualMachine, id string, ch chan<- *proto.StatusUpdate) bool {
	if s.isBootDiskAttached(id, ch) {
		return true
	}

	diskID := s.findCreatedDisk(vm, ch)
	if diskID == "" {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: "No boot disk attached"}
		return false
	}

	return s.attachDisk(id, diskID, ch)
}

func (s *OvirtService) isBootDiskAttached(id string, ch chan<- *proto.StatusUpdate) bool {
	ch <- &proto.StatusUpdate{
		ServiceName: serviceName,
		Message:     "Check if boot disk is attached to VM",
	}

	var attachments DiskAttachments
	err := s.client.GetAndParse(fmt.Sprintf("vms/%s/diskattachments", id), &attachments)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	for _, a := range attachments.Attachments {
		if a.Bootable == true {
			return true
		}
	}

	return false
}

func (s *OvirtService) findCreatedDisk(vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) string {
	var disks Disks
	err := s.client.GetAndParse("/disks?search=number_of_vms=0", &disks)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return ""
	}

	if len(disks.Disks) == 0 {
		return ""
	}

	newest := disks.Disks[0]
	if newest.Name == s.configService.BootDiskName(vm) {
		return newest.ID
	}

	return ""
}

func (s *OvirtService) attachDisk(id, diskID string, ch chan<- *proto.StatusUpdate) bool {
	d := &NewDiskAttachment{
		Bootable:    true,
		PassDiscard: false,
		Interface:   "virtio_scsi",
		Active:      true,
		Disk: Disk{
			ID: diskID,
		},
	}
	ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: "Attaching disk " + diskID, DebugMessage: string(d.serialize())}

	b, err := s.client.SendRequest(fmt.Sprintf("/vms/%s/diskattachments", id), "POST", bytes.NewReader(d.serialize()))
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: "Disk attached", DebugMessage: string(b)}
	return s.waitForVMStatus(id, "down", ch)
}

func (s *OvirtService) startVM(id string, ch chan<- *proto.StatusUpdate) bool {
	body := strings.NewReader("<action/>")
	b, err := s.client.SendRequest(fmt.Sprintf("vms/%s/start", id), "POST", body)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: "VM started", DebugMessage: string(b)}
	return true
}

func (s *OvirtService) deleteVM(id string, ch chan<- *proto.StatusUpdate) bool {
	b, err := s.client.SendRequest(fmt.Sprintf("vms/%s", id), "DELETE", nil)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: "VM deletion initiated", DebugMessage: string(b)}
	return true
}
