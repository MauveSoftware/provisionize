package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/MauveSoftware/provisionize/cmd/provisionize/config"
	"github.com/MauveSoftware/provisionize/dns"
	"github.com/MauveSoftware/provisionize/server"
	"github.com/MauveSoftware/provisionize/vm"
	"github.com/pkg/errors"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const version = "0.1"

var (
	showVersion = kingpin.Flag("version", "Shows version info").Short('v').Bool()
	configFile  = kingpin.Flag("config", "Path to config file").Short('c').Default("config.yml").String()
)

func main() {
	kingpin.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	services := []server.ProvisionService{
		ovirtService(cfg),
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

func ovirtService(cfg *config.Config) server.ProvisionService {
	c := cfg.Ovirt

	template, err := ioutil.ReadFile(c.TemplatePath)
	if err != nil {
		log.Fatal(errors.Wrap(err, "could not load template file"))
	}

	svc, err := vm.NewService(c.URL, c.Username, c.Password, string(template))
	if err != nil {
		log.Fatal(errors.Wrap(err, "could initialize oVirt service"))
	}

	return svc
}

func googleCloudService(cfg *config.Config) server.ProvisionService {
	return &dns.GoogleCloudDNSService{}
}

func printVersion() {
	fmt.Println("Mauve Provisionize")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Daniel Czerwonk")
	fmt.Println("Copyright: Mauve Mailorder Software, 2019. Licensed under MIT license")
}
