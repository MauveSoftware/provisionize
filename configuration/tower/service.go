package tower

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

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
	job, debugInfo, err := s.postStartRequest(fqdn, templateID, ch)
	if err != nil {
		return
	}

	ch <- &proto.StatusUpdate{
		Message:     fmt.Sprintf("Started job %d (%s) for playbook %s", job.ID, job.Name, job.Playbook),
		ServiceName: serviceName,
	}

	return s.waitForJobToComplete(job, ch)
}

func (s *TowerService) postStartRequest(fqdn string, templateID uint, ch chan<- *proto.StatusUpdate) (job *Job, debugInfo string, err error) {
	body := fmt.Sprintf(`{"limit": "%s"}`, fqdn)
	url := fmt.Sprintf("%s/job_templates/%d/launch/", s.baseURL, templateID)

	ch <- &proto.StatusUpdate{
		Message:      fmt.Sprintf("Starting Job with template %d", templateID),
		ServiceName:  serviceName,
		DebugMessage: fmt.Sprintf("URL: %s\nBody: %s", url, body),
	}

	status, b, err := s.sendRequest("POST", url, body)
	if err != nil {
		return
	}

	debugInfo = string(b)

	if status != http.StatusCreated {
		err = errors.New(fmt.Sprintf("could not start job (status code %d)", status))
		return
	}

	job = &Job{}
	err = json.Unmarshal(b, job)
	if err != nil {
		err = errors.Wrapf(err, "could not parse result to job")
		return
	}

	return job, debugInfo, nil
}

func (s *TowerService) waitForJobToComplete(job *Job, ch chan<- *proto.StatusUpdate) (debugMessage string, err error) {
	status := job.Status

	for {
		select {
		case <-time.After(s.waitTimeout):
			return "", errors.New("Operation timed out")
		case <-time.After(s.pollingInterval):
			j, d, err := s.getJobUpdate(job.ID)
			if err != nil {
				return d, errors.Wrap(err, "could not get job status update")
			}

			if status != j.Status {
				status = j.Status
				ch <- &proto.StatusUpdate{
					Message:      fmt.Sprintf("New status: %s", status),
					ServiceName:  serviceName,
					DebugMessage: d,
				}
			}

			if j.Status == "successfull" {
				return d, nil
			}

			if j.Status == "failed" {
				return d, errors.New("Failed running playbook")
			}
		}
	}
}

func (s *TowerService) getJobUpdate(id uint) (job *Job, debugMessage string, err error) {
	url := fmt.Sprintf("%s/jobs/%d", s.baseURL, id)

	status, b, err := s.sendRequest("GET", url, "")
	if err != nil {
		return
	}

	debugMessage = string(b)

	if status != http.StatusOK {
		err = fmt.Errorf("could not get status update for job %d", id)
		return
	}

	job = &Job{}
	err = json.Unmarshal(b, job)
	if err != nil {
		err = errors.Wrapf(err, "could not parse result to job")
		return
	}

	return job, debugMessage, nil
}

func (s *TowerService) sendRequest(method, url string, body string) (status int, b []byte, err error) {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		err = errors.Wrapf(err, "could not create request with URI %s", url)
		return
	}

	req.SetBasicAuth(s.username, s.password)
	req.Header.Set("content-type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	status = resp.StatusCode

	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		err = errors.Wrap(err, "could not read from response")
		return
	}

	return status, b, nil
}
