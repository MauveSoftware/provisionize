package tower

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MauveSoftware/provisionize/api/proto"
)

type mockConfigService struct {
	count uint
}

func (m *mockConfigService) TowerTemplateIDsForVM(vm *proto.VirtualMachine) []uint {
	ids := make([]uint, m.count)
	for i := uint(0); i < m.count; i++ {
		ids[i] = i + 1
	}

	return ids
}

func TestProvision(t *testing.T) {
	tests := []struct {
		name          string
		templateCount uint
		expectedCalls int
		statusCodes   []int
		expectFail    bool
	}{
		{
			name:          "1 successful job",
			templateCount: 1,
			expectedCalls: 1,
			statusCodes:   []int{201},
		},
		{
			name:          "2 jobs, first fail",
			templateCount: 2,
			expectedCalls: 1,
			statusCodes:   []int{500},
			expectFail:    true,
		},
		{
			name:          "2 jobs, last fail",
			templateCount: 2,
			expectedCalls: 2,
			statusCodes:   []int{201, 500},
			expectFail:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			call := 0
			handler := func(w http.ResponseWriter, r *http.Request) {
				call++
				if call > test.expectedCalls {
					t.Fatalf("expected %d http calls, got %d", test.expectedCalls, call)
				}

				w.WriteHeader(test.statusCodes[call-1])
			}

			s := httptest.NewServer(http.HandlerFunc(handler))
			defer s.Close()

			ch := make(chan *proto.StatusUpdate)
			defer close(ch)

			go func() {
				for update := range ch {
					t.Log(update.Message)
				}
			}()

			svc := NewService(s.URL, "test", "foo", &mockConfigService{count: test.templateCount})

			result := svc.Provision(context.Background(), &proto.VirtualMachine{}, ch)
			assert.Equal(t, !test.expectFail, result, "unexpected fail")
		})
	}
}
