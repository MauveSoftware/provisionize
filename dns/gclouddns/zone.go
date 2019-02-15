package gclouddns

import (
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
		log.Infof("%s record for %s already exists: skipping", recType, name)
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
	return err
}
