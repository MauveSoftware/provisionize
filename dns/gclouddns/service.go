package gclouddns

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"

	"github.com/pkg/errors"

	"github.com/MauveSoftware/provisionize/api/proto"
	pdns "github.com/MauveSoftware/provisionize/dns"
	"github.com/MauveSoftware/provisionize/utils"

	"go.opencensus.io/trace"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
)

const serviceName = "Google Cloud DNS"

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

// Provision creates DNS records for the virtual machine
func (s *GoogleCloudDNSService) Provision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.Provision")
	defer span.End()

	if len(vm.Fqdn) == 0 {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: "No FQDN defined => skipping"}
		return true
	}

	zones, err := s.listZones(ctx)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	err = s.ensureHostRecordsExists(ctx, vm, zones, ch)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	err = s.ensurePTRRecordsExists(ctx, vm, zones, ch)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	return true
}

// Deprovision deletes the DNS records for the virtual machine
func (s *GoogleCloudDNSService) Deprovision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.Deprovision")
	defer span.End()

	if len(vm.Fqdn) == 0 {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: "No FQDN defined => skipping"}
		return true
	}

	zones, err := s.listZones(ctx)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	err = s.ensureHostRecordsAbsent(ctx, vm, zones, ch)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	err = s.ensurePTRRecordsAbsent(ctx, vm, zones, ch)
	if err != nil {
		ch <- &proto.StatusUpdate{ServiceName: serviceName, Failed: true, Message: err.Error()}
		return false
	}

	return true
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

func (s *GoogleCloudDNSService) ensureHostRecordsExists(ctx context.Context, vm *proto.VirtualMachine, zones []*dns.ManagedZone, ch chan<- *proto.StatusUpdate) error {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.ensureHostRecordsExists")
	defer span.End()

	z, err := s.zoneForFQDN(vm.Fqdn, zones, ch)
	if err != nil {
		return err
	}

	recs, err := z.records()
	if err != nil {
		return err
	}

	name := s.hostDNSName(vm)

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

func (s *GoogleCloudDNSService) ensureHostRecordsAbsent(ctx context.Context, vm *proto.VirtualMachine, zones []*dns.ManagedZone, ch chan<- *proto.StatusUpdate) error {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.ensureHostRecordsAbsent")
	defer span.End()

	z, err := s.zoneForFQDN(vm.Fqdn, zones, ch)
	if err != nil {
		return err
	}

	recs, err := z.records()
	if err != nil {
		return err
	}

	name := s.hostDNSName(vm)

	err = z.ensureRecordAbsent(name, "A", recs)
	if err != nil {
		return errors.Wrapf(err, "could not remove A record for %s in %s", name, z.name)
	}

	err = z.ensureRecordAbsent(name, "AAAA", recs)
	if err != nil {
		return errors.Wrapf(err, "could not remove AAAA record for %s in %s", name, z.name)
	}

	return nil
}

func (s *GoogleCloudDNSService) hostDNSName(vm *proto.VirtualMachine) string {
	return strings.Trim(vm.Fqdn, ".") + "."
}

func (s *GoogleCloudDNSService) zoneForFQDN(fqdn string, zones []*dns.ManagedZone, ch chan<- *proto.StatusUpdate) (*zone, error) {
	managedZone := s.findZone(fqdn, zones)
	if managedZone == nil {
		return nil, fmt.Errorf("no zone found for %s", fqdn)
	}

	z := &zone{
		name:      managedZone.Name,
		projectID: s.projectID,
		service:   s.service,
		ch:        ch,
	}

	return z, nil
}

func (s *GoogleCloudDNSService) ensurePTRRecordsExists(ctx context.Context, vm *proto.VirtualMachine, zones []*dns.ManagedZone,
	ch chan<- *proto.StatusUpdate) error {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.ensurePTRRecordsExists")
	defer span.End()

	name := s.hostDNSName(vm)
	err := s.ensurePTRRecordExists(vm.Ipv4.Address, name, zones, ch)
	if err != nil {
		return errors.Wrap(err, "could not create PTR record for IPv4")
	}

	err = s.ensurePTRRecordExists(vm.Ipv6.Address, name, zones, ch)
	if err != nil {
		return errors.Wrap(err, "could not create PTR record for IPv6")
	}

	return nil
}

func (s *GoogleCloudDNSService) ensurePTRRecordExists(addr string, value string, zones []*dns.ManagedZone, ch chan<- *proto.StatusUpdate) error {
	ip := net.ParseIP(addr)
	if ip == nil {
		return fmt.Errorf("could not parse IP %s", ip)
	}

	fqdn := pdns.ReverseDomain(ip)
	z, err := s.zoneForFQDN(fqdn, zones, ch)
	if err != nil {
		return err
	}

	recs, err := z.records()
	if err != nil {
		return errors.Wrapf(err, "could not retrieve record set for zone %s", z.name)
	}

	return z.ensureRecordExists(fqdn+".", "PTR", value, recs)
}

func (s *GoogleCloudDNSService) ensurePTRRecordsAbsent(ctx context.Context, vm *proto.VirtualMachine, zones []*dns.ManagedZone,
	ch chan<- *proto.StatusUpdate) error {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.ensurePTRRecordsAbsent")
	defer span.End()

	name := s.hostDNSName(vm)

	ips, err := net.LookupIP(vm.Fqdn)
	if err != nil {
		return errors.Wrapf(err, "could not lookup A and AAAA records for %s", vm.Fqdn)
	}

	for _, ip := range ips {
		s.ensurePTRRecordAbsent(ip, name, zones, ch)
	}

	return nil
}

func (s *GoogleCloudDNSService) ensurePTRRecordAbsent(ip net.IP, value string, zones []*dns.ManagedZone, ch chan<- *proto.StatusUpdate) error {
	if ip == nil {
		return nil
	}

	fqdn := pdns.ReverseDomain(ip)
	z, err := s.zoneForFQDN(fqdn, zones, ch)
	if err != nil {
		return err
	}

	recs, err := z.records()
	if err != nil {
		return errors.Wrapf(err, "could not retrieve record set for zone %s", z.name)
	}

	return z.ensureRecordAbsent(fqdn+".", "PTR", recs)
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
