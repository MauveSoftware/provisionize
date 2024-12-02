package config

import (
	"io"

	yaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
)

// Config represents the configuration
type Config struct {
	ListenAddress   string                `yaml:"listen_address"`
	Proxmox         *ProxmoxConfig        `yaml:"proxmox"`
	GooglecCloudDNS *GoogleCloudDNSConfig `yaml:"gcloud"`
	AnsibleTower    *AnsibleTowerConfig   `yaml:"ansible_tower"`
	Templates       []*ProvisionTemplate  `yaml:"templates"`
}

// ProvisionTemplate represents a set of templates to apply for a certain template defined in VM
type ProvisionTemplate struct {
	Name             string `yaml:"name"`
	AnsibleTemplates []uint `yaml:"ansible_tower"`
}

// ProxmoxConfig represents the Proxmox configuration part
type ProxmoxConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	NodeIP   string `yaml:"node_ip"`
}

// GoogleCloudDNSConfig represents to DNS configuration part
type GoogleCloudDNSConfig struct {
	CredentialsFile string `yaml:"credentials_file"`
	ProjectID       string `yaml:"project_id"`
}

// AnsibleTowerConfig represents the Ansible Tower configuration part
type AnsibleTowerConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Load reads a reader and parses the content
func Load(r io.Reader) (*Config, error) {
	config := &Config{}
	err := yaml.NewDecoder(r).Decode(config)
	if err != nil {
		return nil, errors.Wrap(err, "could parse config")
	}

	return config, nil
}
