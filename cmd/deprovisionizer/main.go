package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/MauveSoftware/provisionize/api/proto"
	"github.com/MauveSoftware/provisionize/clientutils"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"
)

const version = "0.1.0"

var (
	showVersion = kingpin.Flag("version", "Shows version info").Short('v').Bool()
	apiAddress  = kingpin.Flag("api", "API endpoint of the provisionize service").Default("[::1]:1337").String()
	id          = kingpin.Flag("id", "Internal identifier of the VM").String()
	vmName      = kingpin.Arg("name", "Name of the VM to delete").Required().String()
	clusterName = kingpin.Flag("cluster", "Name of the cluster the VM should be removed from").String()
	fqdn        = kingpin.Flag("fqdn", "Full qualified domain name of the VM").Default("").String()
	debug       = kingpin.Flag("debug", "Print debug information recevied from server").Bool()
)

func main() {
	kingpin.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	success, err := startDeprovisioning()
	if err != nil {
		log.Fatal(err)
	}

	if !success {
		os.Exit(1)
	}
}

func printVersion() {
	fmt.Println("Deprovisionizer")
	fmt.Println("CLI client for Mauve Provisionize")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Daniel Czerwonk")
	fmt.Println("Copyright: Mauve Mailorder Software, 2019. Licensed under MIT license")
}

func startDeprovisioning() (bool, error) {
	conn, err := grpc.Dial(*apiAddress, grpc.WithInsecure())
	if err != nil {
		return false, errors.Wrap(err, "could not connect to service")
	}
	defer conn.Close()

	client := proto.NewProvisionizeServiceClient(conn)

	req := requestFromParameters()
	stream, err := client.Deprovisionize(context.Background(), req)
	if err != nil {
		return false, errors.Wrap(err, "error on provisionize call")
	}

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return true, nil
		}

		if err != nil {
			return false, err
		}

		clientutils.LogServiceResult(in, *debug)

		if in.Failed {
			return false, nil
		}
	}
}

func requestFromParameters() *proto.ProvisionizeRequest {
	return &proto.ProvisionizeRequest{
		RequestId: uuid.New().String(),
		VirtualMachine: &proto.VirtualMachine{
			ClusterName: *clusterName,
			Id:          *id,
			Fqdn:        *fqdn,
			Name:        *vmName,
		},
	}
}
