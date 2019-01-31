package config

import (
	"io"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
)

// Config represents the configuration
type Config struct {
	ListenAddress   string                `yaml:"listen_address"`
	Ovirt           *OvirtConfig          `yaml:"ovirt"`
	GooglecCloudDNS *GoogleCloudDNSConfig `yaml:"gcloud"`
}

// OvirtConfig represents to oVirt configuration part
type OvirtConfig struct {
	URL          string `yaml:"url"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	TemplatePath string `yaml:"template_path"`
}

// GoogleCloudDNSConfig represents to DNS configuration part
type GoogleCloudDNSConfig struct {
	ProjectID string `yaml:"project_id"`
}

// Load reads a reader and parses the content
func Load(r io.Reader) (*Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "could not read from reader")
	}

	config := &Config{}
	err = yaml.Unmarshal(b, config)
	if err != nil {
		return nil, errors.Wrap(err, "could parse config")
	}

	return config, nil
}
