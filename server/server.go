package server

import (
	"net"

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

func (srv *server) Provisionize(req *proto.ProvisionVirtualMachineRequest, stream proto.ProvisionizeService_ProvisionizeServer) error {
	log.Info("Received Provisionize request:", req)
	ctx, span := trace.StartSpan(stream.Context(), "API.Provisionize")
	defer span.End()

	// TODO: sanity checks

	for _, s := range srv.services {
		r := s.PerformStep(ctx, req.VirtualMachine)
		err := stream.Send(r)
		if err != nil {
			log.Errorf("Error while sending update to client: %v", err)
			return err
		}

		if r.Failed {
			log.Errorf("Error occured while processing #%s: %v", req.RequestId, r.Message)
			return nil
		}
	}

	return nil
}
