package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/MauveSoftware/provisionize/cmd/provisionize/config"
	"github.com/MauveSoftware/provisionize/configuration/tower"
	"github.com/MauveSoftware/provisionize/dns/gclouddns"
	"github.com/MauveSoftware/provisionize/server"
	"github.com/MauveSoftware/provisionize/vm/ovirt"
	openzipkin "github.com/openzipkin/zipkin-go"
	zipkinHTTP "github.com/openzipkin/zipkin-go/reporter/http"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/exporter/zipkin"
	"go.opencensus.io/trace"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const version = "0.5.0"

var (
	showVersion    = kingpin.Flag("version", "Shows version info").Short('v').Bool()
	configFile     = kingpin.Flag("config", "Path to config file").Short('c').Default("config.yml").String()
	zipkinEndpoint = kingpin.Flag("zipkin-endpoint", "URL to sent tracing information to").String()
)

func main() {
	kingpin.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	initializeZipkin()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	templateManager := newTemplateManager(cfg.Templates)
	services := []server.ProvisionService{
		ovirtService(cfg, templateManager),
		googleCloudService(cfg),
	}

	list, err := net.Listen("tcp", cfg.ListenAddress)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "could not listen on %s", cfg.ListenAddress))
	}

	server.StartServer(list, services)
}

func loadConfig() (*config.Config, error) {
	f, err := os.Open(*configFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not open from config file")
	}
	defer f.Close()

	return config.Load(f)
}

func ovirtService(cfg *config.Config, t *templateManager) server.ProvisionService {
	c := cfg.Ovirt

	template, err := ioutil.ReadFile(c.TemplatePath)
	if err != nil {
		log.Fatal(errors.Wrap(err, "could not load template file"))
	}

	svc, err := ovirt.NewService(c.URL, c.Username, c.Password, string(template), t)
	if err != nil {
		log.Fatal(errors.Wrap(err, "could initialize oVirt service"))
	}

	return svc
}

func googleCloudService(cfg *config.Config) server.ProvisionService {
	f, err := os.Open(cfg.GooglecCloudDNS.CredentialsFile)
	if err != nil {
		log.Fatal(errors.Wrap(err, "could not load Google Cloud credentials file"))
	}
	defer f.Close()

	svc, err := gclouddns.NewDNSService(cfg.GooglecCloudDNS.ProjectID, f)
	if err != nil {
		log.Fatal(errors.Wrap(err, "could initialize Google Cloud DNS service"))
	}

	return svc
}

func ansibleTowerService(cfg *config.Config, t *templateManager) server.ProvisionService {
	return &tower.NewService(cfg.AnsibleTower.URL, cfg.AnsibleTower.Username, cfg.AnsibleTower.Password, t)
}

func initializeZipkin() {
	if len(*zipkinEndpoint) == 0 {
		return
	}

	localEndpoint, err := openzipkin.NewEndpoint("provisionize", ":0")
	if err != nil {
		log.Error(err)
		return
	}

	reporter := zipkinHTTP.NewReporter(*zipkinEndpoint)
	exporter := zipkin.NewExporter(reporter, localEndpoint)
	trace.RegisterExporter(exporter)

	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
}

func printVersion() {
	fmt.Println("Mauve Provisionize")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Daniel Czerwonk")
	fmt.Println("Copyright: Mauve Mailorder Software, 2019. Licensed under MIT license")
}
