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

	done := make(chan bool)
	defer close(done)

	updates := make(chan *proto.StatusUpdate)

	go srv.updateHandler(req.RequestId, stream, updates, done)

	for _, s := range srv.services {
		if !s.Provision(ctx, req.VirtualMachine, updates) {
			break
		}
	}

	close(updates)
	<-done
	return nil
}

func (srv *server) updateHandler(id string, stream proto.ProvisionizeService_ProvisionizeServer, updates chan *proto.StatusUpdate,
	done chan bool) {
	for update := range updates {
		log.Infof("Request: %s\nService: %s\nMessage: %s", id, update.ServiceName, update.Message)

		if len(update.DebugMessage) > 0 {
			log.Debugf("Request: %s\nService: %s\nDebug-Message: %s", id, update.ServiceName, update.DebugMessage)
		}

		err := stream.Send(update)
		if err != nil {
			log.Errorf("Error while sending update to client: %v", err)
		}
	}

	done <- true
}
