package server

import (
	"context"
	"fmt"
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

func (m *mockService) PerformStep(ctx context.Context, vm *proto.VirtualMachine) *proto.ServiceResult {
	m.wasCalled = true

	if m.err != nil {
		return &proto.ServiceResult{Success: false, Message: m.err.Error(), Name: m.name}
	}

	return &proto.ServiceResult{Success: true, Name: m.name}
}

func (m *mockService) verifyExpectation(t *testing.T) {
	assert.Equal(t, m.expectCall, m.wasCalled, m.name+" called?")
}

func TestProvisionize(t *testing.T) {
	tests := []struct {
		name           string
		services       []*mockService
		expectedResult *proto.Result
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
			expectedResult: &proto.Result{
				Success: true,
				ServiceResults: []*proto.ServiceResult{
					{
						Name:    "service1",
						Success: true,
					},
					{
						Name:    "service2",
						Success: true,
					},
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
			expectedResult: &proto.Result{
				Success: false,
				ServiceResults: []*proto.ServiceResult{
					{
						Name:    "service1",
						Success: false,
						Message: "test error",
					},
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
			expectedResult: &proto.Result{
				Success: false,
				ServiceResults: []*proto.ServiceResult{
					{
						Name:    "service1",
						Success: true,
					},
					{
						Name:    "service2",
						Success: false,
						Message: "test error",
					},
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
			res, err := srv.Provisionize(context.Background(), req)
			if err != nil {
				t.Error(err)
			}

			for _, svc := range test.services {
				svc.verifyExpectation(t)
			}

			assert.Equal(t, test.expectedResult, res)
		})
	}
}
