package gclouddns

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net"
	"strings"

	"github.com/MauveSoftware/provisionize/api/proto"
	pdns "github.com/MauveSoftware/provisionize/dns"
	"github.com/MauveSoftware/provisionize/utils"

	"go.opencensus.io/trace"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
)

// GoogleCloudDNSService creates DNS records in Google Cloud DNS
type GoogleCloudDNSService struct {
	service   *dns.Service
	projectID string
}

// NewDNSService creates a new instance of GoogleCloudDNSService
func NewDNSService(projectID string, serviceAccountJSON io.Reader) (*GoogleCloudDNSService, error) {
	b, err := ioutil.ReadAll(serviceAccountJSON)
	if err != nil {
		return nil, errors.Wrap(err, "could not read credentials")
	}

	ctx := context.Background()
	cfg, err := google.JWTConfigFromJSON(b, dns.CloudPlatformScope)
	if err != nil {
		return nil, errors.Wrap(err, "failed get credentials from JSON file")
	}

	service, err := dns.New(cfg.Client(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize DNS service")
	}

	return &GoogleCloudDNSService{
		service:   service,
		projectID: projectID,
	}, nil
}

// PerformStep creates DNS records for the virtual machine
func (s *GoogleCloudDNSService) PerformStep(ctx context.Context, vm *proto.VirtualMachine) *proto.ServiceResult {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.PerformStep")
	defer span.End()

	result := &proto.ServiceResult{
		Name: "Google Cloud DNS",
	}

	if len(vm.Fqdn) == 0 {
		result.Success = true
		result.Message = "No FQDN defined: skipping DNS record creation"
		return result
	}

	zones, err := s.listZones(ctx)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	err = s.ensureHostRecordsExists(ctx, vm, zones)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	err = s.ensurePTRRecordsExists(ctx, vm, zones)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	result.Success = true
	return result
}

func (s *GoogleCloudDNSService) listZones(ctx context.Context) ([]*dns.ManagedZone, error) {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.listZones")
	defer span.End()

	resp, err := s.service.ManagedZones.List(s.projectID).Do()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve list of managed domains")
	}

	return resp.ManagedZones, nil
}

func (s *GoogleCloudDNSService) ensureHostRecordsExists(ctx context.Context, vm *proto.VirtualMachine, zones []*dns.ManagedZone) error {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.ensureHostRecordsExists")
	defer span.End()

	managedZone := s.findZone(vm.Fqdn, zones)
	if managedZone == nil {
		return fmt.Errorf("no zone found for %s", vm.Fqdn)
	}
	z := &zone{
		name:      managedZone.Name,
		projectID: s.projectID,
		service:   s.service,
	}

	recs, err := z.records()
	if err != nil {
		return errors.Wrapf(err, "could not retrieve record set for zone %s", z.name)
	}

	name := strings.Trim(vm.Fqdn, ".") + "."

	err = z.ensureRecordExists(name, "A", vm.Ipv4.Address, recs)
	if err != nil {
		return errors.Wrapf(err, "could not create A record for %s in %s", name, z.name)
	}

	err = z.ensureRecordExists(name, "AAAA", vm.Ipv6.Address, recs)
	if err != nil {
		return errors.Wrapf(err, "could not create AAAA record for %s in %s", name, z.name)
	}

	return nil
}

func (s *GoogleCloudDNSService) ensurePTRRecordsExists(ctx context.Context, vm *proto.VirtualMachine, zones []*dns.ManagedZone) error {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.ensurePTRRecordsExists")
	defer span.End()

	name := strings.Trim(vm.Fqdn, ".") + "."
	err := s.ensurePTRRecordExists(vm.Ipv4.Address, name, zones)
	if err != nil {
		return errors.Wrap(err, "could not create PTR record for IPv4")
	}

	err = s.ensurePTRRecordExists(vm.Ipv6.Address, name, zones)
	if err != nil {
		return errors.Wrap(err, "could not create PTR record for IPv6")
	}

	return nil
}

func (s *GoogleCloudDNSService) ensurePTRRecordExists(addr string, value string, zones []*dns.ManagedZone) error {
	ip := net.ParseIP(addr)
	if ip == nil {
		return fmt.Errorf("could not parse IP %s", ip)
	}

	fqdn := pdns.ReverseDomain(ip)
	managedZone := s.findZone(fqdn, zones)
	if managedZone == nil {
		return fmt.Errorf("no zone found for %s", fqdn)
	}
	z := zone{
		name:      managedZone.Name,
		projectID: s.projectID,
		service:   s.service,
	}

	recs, err := z.records()
	if err != nil {
		return errors.Wrapf(err, "could not retrieve record set for zone %s", z.name)
	}

	return z.ensureRecordExists(fqdn+".", "PTR", value, recs)
}

func (s *GoogleCloudDNSService) findZone(fqdn string, zones []*dns.ManagedZone) *dns.ManagedZone {
	reverseFQDN := utils.ReverseString(fqdn)

	var best *dns.ManagedZone
	bestLen := 0
	for _, z := range zones {
		rev := utils.ReverseString(z.DnsName[:len(z.DnsName)-1])

		if strings.HasPrefix(reverseFQDN, rev) && len(z.DnsName) > bestLen {
			best = z
			bestLen = len(z.DnsName)
		}
	}

	return best
}
