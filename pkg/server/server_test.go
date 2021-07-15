package server

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc"

	"github.com/stretchr/testify/assert"

	"github.com/MauveSoftware/provisionize/pkg/api/proto"
)

type mockService struct {
	name string
	err  error
}

func (m *mockService) Provision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	status := &proto.StatusUpdate{
		ServiceName: m.name,
	}
	result := true

	if m.err != nil {
		status.Failed = true
		status.Message = m.err.Error()
		result = false
	}

	ch <- status
	return result
}

func (m *mockService) Deprovision(ctx context.Context, vm *proto.VirtualMachine, ch chan<- *proto.StatusUpdate) bool {
	status := &proto.StatusUpdate{
		ServiceName: m.name,
	}
	result := true

	if m.err != nil {
		status.Failed = true
		status.Message = m.err.Error()
		result = false
	}

	ch <- status
	return result
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
					name: "service1",
				},
				&mockService{
					name: "service2",
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
					name: "service1",
					err:  fmt.Errorf("test error"),
				},
				&mockService{
					name: "service2",
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
					name: "service1",
				},
				&mockService{
					name: "service2",
					err:  fmt.Errorf("test error"),
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

			req := &proto.ProvisionizeRequest{}
			stream := &mockStream{}
			err := srv.Provisionize(req, stream)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, test.expectedResult, stream.updates)
		})
	}
}

func TestDeprovisionize(t *testing.T) {
	tests := []struct {
		name           string
		services       []*mockService
		expectedResult []*proto.StatusUpdate
	}{
		{
			name: "2 services",
			services: []*mockService{
				&mockService{
					name: "service1",
				},
				&mockService{
					name: "service2",
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
					name: "service1",
					err:  fmt.Errorf("test error"),
				},
				&mockService{
					name: "service2",
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
					name: "service1",
				},
				&mockService{
					name: "service2",
					err:  fmt.Errorf("test error"),
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

			req := &proto.ProvisionizeRequest{}
			stream := &mockStream{}
			err := srv.Deprovisionize(req, stream)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, test.expectedResult, stream.updates)
		})
	}
}
