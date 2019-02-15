package gclouddns

import (
	"context"
	"fmt"
	"github.com/MauveSoftware/provisionize/utils"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net"
	"strings"

	"github.com/MauveSoftware/provisionize/api/proto"

	"go.opencensus.io/trace"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
)

type GoogleCloudDNSService struct {
	service   *dns.Service
	projectID string
}

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
func (s *GoogleCloudDNSService) PerformStep(ctx context.Context, vm *proto.VirtualMachine) error {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.PerformStep")
	defer span.End()

	zones, err := s.listZones(ctx)
	if err != nil {
		return err
	}

	err = s.ensureHostRecordsExists(ctx, vm, zones)
	if err != nil {
		return err
	}

	return err
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

func (s *GoogleCloudDNSService) ensurePTRRecordExists(ctx context.Context, ip net.IP, value string) error {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.ensurePTRRecordExists")
	defer span.End()

	// TODO: implement me!

	return nil
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
