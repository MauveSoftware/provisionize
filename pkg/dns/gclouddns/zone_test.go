package gclouddns

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/dns/v1"
)

func TestFindRecordSet(t *testing.T) {
	recs := []*dns.ResourceRecordSet{
		{Name: "host1.example.com.", Type: "A", Rrdatas: []string{"10.0.0.1"}},
		{Name: "host1.example.com.", Type: "AAAA", Rrdatas: []string{"::1"}},
		{Name: "host2.example.com.", Type: "A", Rrdatas: []string{"10.0.0.2"}},
	}

	tests := []struct {
		name        string
		recName     string
		recType     string
		expectFound bool
		expectedIdx int
	}{
		{
			name:        "exact match",
			recName:     "host1.example.com.",
			recType:     "A",
			expectFound: true,
			expectedIdx: 0,
		},
		{
			name:        "same name, different type",
			recName:     "host1.example.com.",
			recType:     "AAAA",
			expectFound: true,
			expectedIdx: 1,
		},
		{
			name:        "same type, different name",
			recName:     "host3.example.com.",
			recType:     "A",
			expectFound: false,
		},
		{
			name:        "no records at all",
			recName:     "host1.example.com.",
			recType:     "MX",
			expectFound: false,
		},
	}

	t.Parallel()

	z := &zone{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec, found := z.findRecordSet(test.recName, test.recType, recs)

			assert.Equal(t, test.expectFound, found, "found flag for %s did not match expectation", test.name)
			if test.expectFound {
				assert.Same(t, recs[test.expectedIdx], rec, "returned record for %s did not match expectation", test.name)
			} else {
				assert.Nil(t, rec)
			}
		})
	}
}
