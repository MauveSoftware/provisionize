package gclouddns

import (
	"encoding/json"
	"fmt"

	"github.com/MauveSoftware/provisionize/pkg/api/proto"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/dns/v1"
)

const (
	defaultTTL = 300
)

type zone struct {
	service   *dns.Service
	projectID string
	name      string
	ch        chan<- *proto.StatusUpdate
}

func (z *zone) records() ([]*dns.ResourceRecordSet, error) {
	resp, err := z.service.ResourceRecordSets.List(z.projectID, z.name).Do()
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve record set for zone %s", z.name)
	}

	return resp.Rrsets, nil
}

func (z *zone) ensureRecordExists(name, recType, value string, recs []*dns.ResourceRecordSet) error {
	_, found := z.findRecordSet(name, recType, recs)
	if found {
		message := fmt.Sprintf("%s record for %s already exists: skipping", recType, name)
		log.Info(message)
		z.ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: message}

		return nil
	}

	return z.createRecord(name, recType, value)
}

func (z *zone) ensureRecordAbsent(name, recType string, recs []*dns.ResourceRecordSet) error {
	rec, found := z.findRecordSet(name, recType, recs)
	if !found {
		message := fmt.Sprintf("%s record for %s already removed: skipping", recType, name)
		log.Info(message)
		z.ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: message}

		return nil
	}

	return z.removeRecord(rec)
}

func (z *zone) findRecordSet(name, recType string, recs []*dns.ResourceRecordSet) (record *dns.ResourceRecordSet, found bool) {
	for _, rec := range recs {
		if rec.Type == recType && rec.Name == name {
			return rec, true
		}
	}

	return nil, false
}

func (z *zone) createRecord(name, recType, value string) error {
	log.Infof("Creating %s record for %s with value %s", recType, name, value)

	record := &dns.ResourceRecordSet{
		Type:    recType,
		Name:    name,
		Ttl:     int64(defaultTTL),
		Rrdatas: []string{value},
	}

	change := &dns.Change{
		Additions: []*dns.ResourceRecordSet{record},
	}

	c, err := z.service.Changes.Create(z.projectID, z.name, change).Do()
	if err == nil {
		b, _ := json.Marshal(c)
		z.ch <- &proto.StatusUpdate{
			ServiceName:  serviceName,
			Message:      fmt.Sprintf("Created: %s\t%d\t%s\t%s", record.Name, record.Ttl, record.Type, record.Rrdatas[0]),
			DebugMessage: string(b),
		}
	}

	return err
}

func (z *zone) removeRecord(record *dns.ResourceRecordSet) error {
	log.Infof("Deleting %s record for %s with value %s", record.Type, record.Name, record.Rrdatas)

	change := &dns.Change{
		Deletions: []*dns.ResourceRecordSet{record},
	}

	c, err := z.service.Changes.Create(z.projectID, z.name, change).Do()
	if err == nil {
		b, _ := json.Marshal(c)
		z.ch <- &proto.StatusUpdate{
			ServiceName:  serviceName,
			Message:      fmt.Sprintf("Deleted: %s\t%d\t%s\t%s", record.Name, record.Ttl, record.Type, record.Rrdatas[0]),
			DebugMessage: string(b),
		}
	}

	return err
}
