package gclouddns

import (
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/dns/v1"
	"testing"
)

func TestFindZoneForFQDN(t *testing.T) {
	tests := []struct {
		name          string
		fqdn          string
		zones         []*dns.ManagedZone
		expectMatch   bool
		expectedIndex int
	}{
		{
			name:        "empty list",
			fqdn:        "abc.routing.rocks",
			zones:       []*dns.ManagedZone{},
			expectMatch: false,
		},
		{
			name: "not in list",
			fqdn: "abc.routing.rocks",
			zones: []*dns.ManagedZone{
				{
					DnsName: "mauve.de.",
				},
			},
			expectMatch: false,
		},
		{
			name: "2 matches in list",
			fqdn: "abc.dus.routing.rocks",
			zones: []*dns.ManagedZone{
				{
					DnsName: "mauve.de.",
				},
				{
					DnsName: "dus.routing.rocks.",
				},
				{
					DnsName: "routing.rocks.",
				},
			},
			expectMatch:   true,
			expectedIndex: 1,
		},
	}

	t.Parallel()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &GoogleCloudDNSService{}
			z := service.findZoneForFQDN(test.fqdn, test.zones)

			if test.expectMatch {
				assert.Exactly(t, test.zones[test.expectedIndex], z)
			} else {
				assert.Nil(t, z)
			}
		})
	}
}
