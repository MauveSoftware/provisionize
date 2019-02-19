package tower

import (
	"context"

	"github.com/MauveSoftware/provisionize/api/proto"

	"go.opencensus.io/trace"
)

// TowerService is the service responsible for configuring the VM by using ansible tower
type TowerService struct {
	url           string
	username      string
	password      string
	configService *ConfigService
}

// NewService returns a new instance of TowerService
func NewService(url, username, password string, configService *ConfigService) *TowerService {
	return &TowerService{
		url:           url,
		username:      username,
		password:      password,
		configService: configService,
	}
}

// Provision performs the required ansible playbook
func (s *TowerService) Provision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	ctx, span := trace.StartSpan(ctx, "TowerService.Provision")
	defer span.End()

	// POST https://awx.m-eshop.de/api/v2/job_templates/25/launch

	/*
		{
			"limit": "foo"
		}
	*/

	return true
}
