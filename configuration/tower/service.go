package tower

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/MauveSoftware/provisionize/api/proto"

	"go.opencensus.io/trace"
)

const (
	serviceName       = "Ansible Tower"
	createdStatusCode = 201
)

// TowerService is the service responsible for configuring the VM by using ansible tower
type TowerService struct {
	baseURL       string
	username      string
	password      string
	configService ConfigService
	client        *http.Client
}

// NewService returns a new instance of TowerService
func NewService(url, username, password string, configService ConfigService) *TowerService {
	return &TowerService{
		baseURL:       completeAPIURL(url),
		username:      username,
		password:      password,
		configService: configService,
		client:        &http.Client{},
	}
}

func completeAPIURL(url string) string {
	return strings.TrimRight(url, "/") + "/api/v2"
}

// Provision performs the required ansible playbook
func (s *TowerService) Provision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	ctx, span := trace.StartSpan(ctx, "TowerService.Provision")
	defer span.End()

	for _, id := range s.configService.TowerTemplateIDsForVM(vm) {
		debugInfo, err := s.startJob(vm.Fqdn, id, ch)
		if err != nil {
			ch <- &proto.StatusUpdate{
				Failed:       true,
				Message:      err.Error(),
				ServiceName:  serviceName,
				DebugMessage: debugInfo,
			}
			return false
		}
	}

	return true
}

func (s *TowerService) startJob(fqdn string, templateID uint, ch chan<- *proto.StatusUpdate) (debugInfo string, err error) {
	body := fmt.Sprintf(`{limit="%s"}`, fqdn)
	url := fmt.Sprintf("%s/job_templates/%d/launch", s.baseURL, templateID)

	ch <- &proto.StatusUpdate{
		Message:      fmt.Sprintf("Starting Job with template %d", templateID),
		ServiceName:  serviceName,
		DebugMessage: fmt.Sprintf("URL: %s\nBody: %s", url, body),
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		err = errors.Wrapf(err, "could not create job request for template %d", templateID)
		return
	}
	req.SetBasicAuth(s.username, s.password)

	resp, err := s.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = errors.Wrap(err, "could not read from response")
		return
	}

	if resp.StatusCode != 201 {
		debugInfo = string(b)
		err = errors.New(fmt.Sprintf("could not start job (status code %d)", resp.StatusCode))
		return
	}

	return "", nil
}
