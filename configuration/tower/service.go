package tower

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/MauveSoftware/provisionize/api/proto"

	"go.opencensus.io/trace"
)

const (
	serviceName = "Ansible Tower"
)

// TowerService is the service responsible for configuring the VM by using ansible tower
type TowerService struct {
	baseURL         string
	username        string
	password        string
	configService   ConfigService
	client          *http.Client
	waitTimeout     time.Duration
	pollingInterval time.Duration
}

type apiResponse struct {
	statusCode int
	body       []byte
}

type jobFuncResult struct {
	err          error
	debugMessage string
	job          *Job
}

// NewService returns a new instance of TowerService
func NewService(url, username, password string, configService ConfigService) *TowerService {
	return &TowerService{
		baseURL:         completeAPIURL(url),
		username:        username,
		password:        password,
		configService:   configService,
		client:          &http.Client{},
		waitTimeout:     2 * time.Minute,
		pollingInterval: 10 * time.Second,
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
		debugInfo, err := s.startJob(vm, id, ch)
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

// Deprovision is called when VM is beeing removed
func (s *TowerService) Deprovision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	// nothing to clean up in Tower
	return true
}

func (s *TowerService) startJob(vm *proto.VirtualMachine, templateID uint, ch chan<- *proto.StatusUpdate) (debugInfo string, err error) {
	res := s.postStartRequest(vm, templateID, ch)
	if res.err != nil {
		return res.debugMessage, res.err
	}
	job := res.job

	ch <- &proto.StatusUpdate{
		Message:      fmt.Sprintf("Start job %d (%s) for playbook %s", job.ID, job.Name, job.Playbook),
		ServiceName:  serviceName,
		DebugMessage: res.debugMessage,
	}

	d, err := s.waitForJobToComplete(job, ch)
	if err != nil {
		return d, err
	}

	return d, s.pushStdOut(job.ID, ch)
}

func (s *TowerService) postStartRequest(vm *proto.VirtualMachine, templateID uint, ch chan<- *proto.StatusUpdate) *jobFuncResult {
	body := fmt.Sprintf(`{"limit": "%s", "extra_vars": "ansible_ssh_host: %s"}`, vm.Fqdn, vm.Ipv4.Address)
	url := fmt.Sprintf("%s/job_templates/%d/launch/", s.baseURL, templateID)

	ch <- &proto.StatusUpdate{
		Message:      fmt.Sprintf("Starting Job with template %d", templateID),
		ServiceName:  serviceName,
		DebugMessage: fmt.Sprintf("URL: %s\nBody: %s", url, body),
	}

	res, err := s.sendRequest("POST", url, "application/json", body)
	if err != nil {
		return &jobFuncResult{err: err}
	}

	if res.statusCode != http.StatusCreated {
		return &jobFuncResult{
			debugMessage: string(res.body),
			err:          errors.New(fmt.Sprintf("could not start job (status code %d)", res.statusCode)),
		}
	}

	job := &Job{}
	err = json.Unmarshal(res.body, job)
	if err != nil {
		return &jobFuncResult{
			debugMessage: string(res.body),
			err:          errors.Wrapf(err, "could not parse result to job"),
		}
	}

	return &jobFuncResult{job: job, debugMessage: string(res.body)}
}

func (s *TowerService) waitForJobToComplete(job *Job, ch chan<- *proto.StatusUpdate) (debugMessage string, err error) {
	status := job.Status

	for {
		select {
		case <-time.After(s.waitTimeout):
			return "", errors.New("Operation timed out")
		case <-time.After(s.pollingInterval):
			res := s.getJobUpdate(job.ID)
			if res.err != nil {
				return res.debugMessage, errors.Wrap(res.err, "could not get job status update")
			}

			if status != res.job.Status {
				status = res.job.Status
				ch <- &proto.StatusUpdate{
					Message:      fmt.Sprintf("New status: %s", status),
					ServiceName:  serviceName,
					DebugMessage: res.debugMessage,
				}
			}

			if res.job.Status == "successful" {
				return res.debugMessage, nil
			}

			if res.job.Status == "failed" {
				s.pushStdOut(res.job.ID, ch)
				return res.debugMessage, errors.New("Failed running playbook")
			}
		}
	}
}

func (s *TowerService) getJobUpdate(id uint) *jobFuncResult {
	url := fmt.Sprintf("%s/jobs/%d", s.baseURL, id)

	res, err := s.sendRequest("GET", url, "application/json", "")
	if err != nil {
		return &jobFuncResult{err: err}
	}

	if res.statusCode != http.StatusOK {
		return &jobFuncResult{
			debugMessage: string(res.body),
			err:          fmt.Errorf("could not get status update for job %d", id),
		}
	}

	job := &Job{}
	err = json.Unmarshal(res.body, job)
	if err != nil {
		return &jobFuncResult{
			debugMessage: string(res.body),
			err:          errors.Wrapf(err, "could not parse result to job"),
		}
	}

	return &jobFuncResult{job: job, debugMessage: string(res.body)}
}

func (s *TowerService) pushStdOut(id uint, ch chan<- *proto.StatusUpdate) error {
	url := fmt.Sprintf("%s/jobs/%d/stdout?format=txt", s.baseURL, id)

	res, err := s.sendRequest("GET", url, "text/plain", "")
	if err != nil {
		return errors.Wrap(err, "could not retrieve output for job")
	}

	ch <- &proto.StatusUpdate{
		ServiceName: serviceName,
		Message:     string(res.body),
	}

	return nil
}

func (s *TowerService) sendRequest(method, url, contentType, body string) (*apiResponse, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return nil, errors.Wrapf(err, "could not create request with URI %s", url)
	}

	req.SetBasicAuth(s.username, s.password)
	req.Header.Set("content-type", contentType)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read from response")
	}

	return &apiResponse{body: b, statusCode: resp.StatusCode}, nil
}
