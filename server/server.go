package server

import (
	"context"
	"net"

	"github.com/MauveSoftware/provisionize/api"
	"github.com/MauveSoftware/provisionize/api/proto"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	log "github.com/sirupsen/logrus"
)

type server struct {
	services []ProvisionService
}

// StartServer starts an gRPC API endpoint
func StartServer(conn net.Listener, services []ProvisionService) error {
	srv := &server{
		services: services,
	}

	s := grpc.NewServer()
	proto.RegisterProvisionizeServiceServer(s, srv)
	reflection.Register(s)

	log.Println("Starting API server on", conn.Addr())
	if err := s.Serve(conn); err != nil {
		return errors.Wrap(err, "failed to serve")
	}

	return nil
}

func (srv *server) Provisionize(ctx context.Context, req *proto.ProvisionVirtualMachineRequest) (*proto.Result, error) {
	log.Info("Received Provisionize request:", req)
	ctx, span := trace.StartSpan(ctx, "API.Provisionize")
	defer span.End()

	// TODO: sanity checks

	var err error
	for _, s := range srv.services {
		err = s.PerformStep(ctx, req.VirtualMachine)
		if err != nil {
			log.Errorf("Error occured while processing #%s: %v", req.RequestId, err)
			return &proto.Result{
				Code:    api.StatusCodeProcessingError,
				Message: err.Error(),
			}, nil
		}
	}

	return &proto.Result{Code: api.StatusCodeOK}, nil
}
