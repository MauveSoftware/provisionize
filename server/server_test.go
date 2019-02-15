package server

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MauveSoftware/provisionize/api/proto"
)

type mockService struct {
	name       string
	expectCall bool
	wasCalled  bool
	err        error
}

func (m *mockService) PerformStep(ctx context.Context, vm *proto.VirtualMachine) *proto.StatusUpdate {
	m.wasCalled = true
	status := &proto.StatusUpdate{
		ServiceName: m.name,
	}

	if m.err != nil {
		status.Failed = true
		status.Message = m.err.Error()
	}

	return status
}

func (m *mockService) verifyExpectation(t *testing.T) {
	assert.Equal(t, m.expectCall, m.wasCalled, m.name+" called?")
}

type mockStream struct {
	grpc.ServerStream
	updates []*proto.StatusUpdate
}

func (s *mockStream) Send(update *proto.StatusUpdate) error {
	s.updates = append(s.updates, update)
	return nil
}

func (s *mockStream) Context() context.Context {
	return context.Background()
}

func TestProvisionize(t *testing.T) {
	tests := []struct {
		name           string
		services       []*mockService
		expectedResult []*proto.StatusUpdate
	}{
		{
			name: "2 services",
			services: []*mockService{
				&mockService{
					name:       "service1",
					expectCall: true,
				},
				&mockService{
					name:       "service2",
					expectCall: true,
				},
			},
			expectedResult: []*proto.StatusUpdate{
				{
					ServiceName: "service1",
				},
				{
					ServiceName: "service2",
				},
			},
		},
		{
			name: "2 services, error on first",
			services: []*mockService{
				&mockService{
					name:       "service1",
					expectCall: true,
					err:        fmt.Errorf("test error"),
				},
				&mockService{
					name:       "service2",
					expectCall: false,
				},
			},
			expectedResult: []*proto.StatusUpdate{
				{
					ServiceName: "service1",
					Failed:      true,
					Message:     "test error",
				},
			},
		},
		{
			name: "2 services, error on second",
			services: []*mockService{
				&mockService{
					name:       "service1",
					expectCall: true,
				},
				&mockService{
					name:       "service2",
					expectCall: true,
					err:        fmt.Errorf("test error"),
				},
			},
			expectedResult: []*proto.StatusUpdate{
				{
					ServiceName: "service1",
				},
				{
					ServiceName: "service2",
					Failed:      true,
					Message:     "test error",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			services := make([]ProvisionService, len(test.services))
			for i, svc := range test.services {
				services[i] = svc
			}

			srv := &server{
				services: services,
			}

			req := &proto.ProvisionVirtualMachineRequest{}
			stream := &mockStream{}
			err := srv.Provisionize(req, stream)
			if err != nil {
				t.Error(err)
			}

			for _, svc := range test.services {
				svc.verifyExpectation(t)
			}

			assert.Equal(t, test.expectedResult, stream.updates)
		})
	}
}
