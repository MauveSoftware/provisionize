package ovirt

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/MauveSoftware/provisionize/api/proto"
	ovirt "github.com/czerwonk/ovirt_api/api"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

const serviceName = "oVirt"

// OvirtService is the service responsible for creating the virtual machine
type OvirtService struct {
	template        string
	client          *ovirt.Client
	waitTimeout     time.Duration
	pollingInterval time.Duration
}

// NewService creates a new instance of OvirtService
func NewService(url, user, pass string, template string) (*OvirtService, error) {
	client, err := ovirt.NewClient(url, user, pass, ovirt.WithDebug())
	if err != nil {
		return nil, errors.Wrap(err, "could not create new oVirt client")
	}

	svc := &OvirtService{
		client:          client,
		template:        template,
		waitTimeout:     2 * time.Minute,
		pollingInterval: 10 * time.Second,
	}

	return svc, nil
}

// PerformStep creates the virtual machine
func (s *OvirtService) PerformStep(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	ctx, span := trace.StartSpan(ctx, "OvirtService.PerformStep")
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
		s.startVM(v.ID, ch) &&
		s.waitForVMStatus(v.ID, "up", ch)
}

func (s *OvirtService) createVM(vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) ([]byte, error) {
	body, err := s.getVMCreateRequest(vm)
	if err != nil {
		return nil, err
	}

	log.Infof("Request for VM %s:\n%s", vm.Name, body)
	ch <- &proto.StatusUpdate{
		ServiceName:  serviceName,
		DebugMessage: body.String(),
		Message:      "Start creating VM",
	}

	b, err := s.client.SendRequest("vms?clone=true", "POST", body)
	if err != nil {
		return nil, err
	}

	log.Infof("Response for VM %s:\n%s", vm.Name, string(b))
	ch <- &proto.StatusUpdate{
		ServiceName:  serviceName,
		DebugMessage: string(b),
		Message:      "VM created successfully",
	}

	return b, nil
}

func (s *OvirtService) getVMCreateRequest(vm *proto.VirtualMachine) (*bytes.Buffer, error) {
	funcs := template.FuncMap{
		"mb_to_byte": func(x uint32) uint64 {
			return uint64(x) * (1 << 20)
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

func (s *OvirtService) getVM(id string) (*VM, error) {
	var vm VM
	err := s.client.GetAndParse(fmt.Sprintf("vms/%s", id), &vm)
	if err != nil {
		return nil, err
	}

	return &vm, nil
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
