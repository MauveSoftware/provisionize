package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"

	"github.com/MauveSoftware/provisionize/api"
	"github.com/MauveSoftware/provisionize/api/proto"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const version = "0.1"

var (
	showVersion  = kingpin.Flag("version", "Shows version info").Short('v').Bool()
	apiAddress   = kingpin.Arg("api", "API endpoint of the provisionize service").String()
	id           = kingpin.Arg("id", "Internal identifier of the VM").String()
	vmName       = kingpin.Arg("name", "Name of the VM to create").String()
	clusterName  = kingpin.Arg("cluster", "Name of the cluster the VM should be deployed on").String()
	templateName = kingpin.Arg("template", "Name of the template to use").String()
	ipv4         = kingpin.Arg("ipv4", "IPv4 address").IP()
	ipv6         = kingpin.Arg("ipv6", "IPv6 address").IP()
	cores        = kingpin.Arg("cores", "Number of CPU cores").Default("4").Uint()
	memory       = kingpin.Arg("memory", "Memory in MB").Default("1024").Uint()
)

func main() {
	kingpin.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	err := startProvisioning()
	if err != nil {
		log.Fatal(err)
	}
}

func printVersion() {
	fmt.Println("Provisionizer")
	fmt.Println("CLI client for Mauve Provisionize")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Daniel Czerwonk")
	fmt.Println("Copyright: Mauve Mailorder Software, 2019. Licensed under MIT license")
}

func startProvisioning() error {
	conn, err := grpc.Dial(*apiAddress, grpc.WithInsecure())
	if err != nil {
		return errors.Wrap(err, "could not connect to service")
	}
	defer conn.Close()

	client := proto.NewProvisionizeServiceClient(conn)

	req := requestFromParameters()
	res, err := client.Provisionize(context.Background(), req)
	if err != nil {
		return errors.Wrap(err, "error on provisionize call")
	}

	if res.Code != api.StatusCodeOK {
		return fmt.Errorf("error: %s", res.Message)
	}

	return nil
}

func requestFromParameters() *proto.ProvisionVirtualMachineRequest {
	return &proto.ProvisionVirtualMachineRequest{
		RequestId: uuid.New().String(),
		VirtualMachine: &proto.VirtualMachine{
			ClusterName:   *clusterName,
			CpuCores:      uint32(*cores),
			Id:            *id,
			Ipv4Addresses: []string{(*ipv4).String()},
			Ipv6Addresses: []string{(*ipv6).String()},
			MemoryMb:      uint32(*memory),
			Name:          *vmName,
			TemplateName:  *templateName,
		},
	}
}
