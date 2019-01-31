package dns

import (
	"context"

	"github.com/MauveSoftware/provisionize/api/proto"
	"go.opencensus.io/trace"
	_ "golang.org/x/oauth2/google"
	_ "google.golang.org/api/dns/v1"
)

type GoogleCloudDNSService struct {
}

func (s *GoogleCloudDNSService) PerformStep(ctx context.Context, vm *proto.VirtualMachine) error {
	ctx, span := trace.StartSpan(ctx, "GoogleCloudDNSService.PerformStep")
	defer span.End()

	// TODO: create A, AAAA and PTR records

	return nil
}
