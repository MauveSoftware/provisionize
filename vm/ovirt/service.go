package ovirt

import (
	"bytes"
	"context"
	"io"
	"text/template"

	"github.com/MauveSoftware/provisionize/api/proto"
	ovirt "github.com/czerwonk/ovirt_api/api"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// OvirtService is the service responsible for creating the virtual machine
type OvirtService struct {
	template string
	client   *ovirt.Client
}

// NewService creates a new instance of OvirtService
func NewService(url, user, pass string, template string) (*OvirtService, error) {
	client, err := ovirt.NewClient(url, user, pass, ovirt.WithDebug())
	if err != nil {
		return nil, errors.Wrap(err, "could not create new oVirt client")
	}

	svc := &OvirtService{
		client:   client,
		template: template,
	}

	return svc, nil
}

// PerformStep creates the virtual machine
func (s *OvirtService) PerformStep(ctx context.Context, vm *proto.VirtualMachine) *proto.ServiceResult {
	ctx, span := trace.StartSpan(ctx, "OvirtService.PerformStep")
	defer span.End()

	result := &proto.ServiceResult{
		Name: "oVirt",
	}

	body, err := s.getVMCreateRequest(vm)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	log.Infof("Request for VM %s:\n%s", vm.Name, body)

	b, err := s.client.SendRequest("vms?clone=true", "POST", body)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	log.Infof("Response for VM %s:\n%s", vm.Name, string(b))
	result.DebugMessage = string(b)

	result.Success = true
	return result
}

func (s *OvirtService) getVMCreateRequest(vm *proto.VirtualMachine) (io.Reader, error) {
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