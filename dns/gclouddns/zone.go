package gclouddns

import (
	"fmt"
	"github.com/MauveSoftware/provisionize/api/proto"
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
		return nil, err
	}

	return resp.Rrsets, nil
}

func (z *zone) ensureRecordExists(name, recType, value string, recs []*dns.ResourceRecordSet) error {
	if z.isInRecordSet(name, recType, recs) {
		message := fmt.Sprintf("%s record for %s already exists: skipping", recType, name)
		log.Infof(message)
		z.ch <- &proto.StatusUpdate{ServiceName: serviceName, Message: message}

		return nil
	}

	return z.createRecord(name, recType, value)
}

func (z *zone) isInRecordSet(name, recType string, recs []*dns.ResourceRecordSet) bool {
	for _, rec := range recs {
		if rec.Type == recType && rec.Name == name {
			return true
		}
	}

	return false
}

func (z *zone) createRecord(name, recType, value string) error {
	log.Infof("Creating %s record for %s with value %s", recType, name, value)
	change := &dns.Change{
		Additions: []*dns.ResourceRecordSet{
			&dns.ResourceRecordSet{
				Type:    recType,
				Name:    name,
				Ttl:     int64(defaultTTL),
				Rrdatas: []string{value},
			},
		},
	}

	_, err := z.service.Changes.Create(z.projectID, z.name, change).Do()
	if err == nil {
		z.ch <- &proto.StatusUpdate{
			ServiceName: serviceName,
			Message:     fmt.Sprintf("Created %s record for %s with value %s in zone %s", recType, name, value, z.name),
		}
	}

	return err
}
