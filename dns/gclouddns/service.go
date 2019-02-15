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

	log "github.com/sirupsen/logrus"
)

const (
	DefaultTTL = 300
)

type GoogleCloudDNSService struct {
	service   *dns.Service
	projectID string
	ttl       uint32
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
		ttl:       DefaultTTL,
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

	z := s.findZoneForFQDN(vm.Fqdn, zones)
	if z == nil {
		return fmt.Errorf("no zone found for %s", vm.Fqdn)
	}

	recs, err := s.service.ResourceRecordSets.List(s.projectID, z.Name).Do()
	if err != nil {
		return errors.Wrapf(err, "could not retrieve record set for zone %s", z.Name)
	}

	name := strings.Trim(vm.Fqdn, ".") + "."

	err = s.ensureRecordExists(name, "A", vm.Ipv4.Address, z, recs.Rrsets)
	if err != nil {
		return errors.Wrapf(err, "could not create A record for %s in %s", name, z.Name)
	}

	err = s.ensureRecordExists(name, "AAAA", vm.Ipv6.Address, z, recs.Rrsets)
	if err != nil {
		return errors.Wrapf(err, "could not create AAAA record for %s in %s", name, z.Name)
	}

	return nil
}

func (s *GoogleCloudDNSService) ensureRecordExists(name, recType, value string, zone *dns.ManagedZone, recs []*dns.ResourceRecordSet) error {
	if s.isInRecordSet(name, recType, recs) {
		log.Infof("%s record for %s already exists: skipping", recType, name)
		return nil
	}

	return s.createRecord(name, recType, value, zone)
}

func (s *GoogleCloudDNSService) findZoneForFQDN(fqdn string, zones []*dns.ManagedZone) *dns.ManagedZone {
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

func (s *GoogleCloudDNSService) ensurePTRRecordExists(ctx context.Context, ip net.IP, value string) error {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.ensurePTRRecordExists")
	defer span.End()

	// TODO: implement me!

	return nil
}

func (s *GoogleCloudDNSService) findZoneForIP(ip net.IP, zones []*dns.ManagedZone) *dns.ManagedZone {
	return nil
}

func (s *GoogleCloudDNSService) isInRecordSet(name, recType string, recs []*dns.ResourceRecordSet) bool {
	for _, rec := range recs {
		if rec.Type == recType && rec.Name == name {
			return true
		}
	}

	return false
}

func (s *GoogleCloudDNSService) createRecord(name, recType, value string, zone *dns.ManagedZone) error {
	log.Infof("Creating %s record for %s with value %s", recType, name, value)
	change := &dns.Change{
		Additions: []*dns.ResourceRecordSet{
			&dns.ResourceRecordSet{
				Type:    recType,
				Name:    name,
				Ttl:     int64(s.ttl),
				Rrdatas: []string{value},
			},
		},
	}

	_, err := s.service.Changes.Create(s.projectID, zone.Name, change).Do()
	return err
}
