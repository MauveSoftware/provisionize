package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	config := `listen_address: "[::]:1337"
proxmox:
  url: https://proxmox:8006/api2/json
  username: pve_prov
  password: pve_pass
gcloud:
  credentials_file: "/config/cred.json"
  project_id: "123"
ansible_tower:
  url: https://tower
  username: ansible
  password: magic
templates:
  - name: linux
    ansible_tower:
      - 1
      - 2
    boot_disk_name: new-disk
`
	expected := &Config{
		ListenAddress: "[::]:1337",
		Proxmox: &ProxmoxConfig{
			URL:      "https://proxmox:8006/api2/json",
			Username: "pve_prov",
			Password: "pve_pass",
		},
		GooglecCloudDNS: &GoogleCloudDNSConfig{
			CredentialsFile: "/config/cred.json",
			ProjectID:       "123",
		},
		AnsibleTower: &AnsibleTowerConfig{
			URL:      "https://tower",
			Username: "ansible",
			Password: "magic",
		},
		Templates: []*ProvisionTemplate{
			{
				Name:             "linux",
				AnsibleTemplates: []uint{1, 2},
			},
		},
	}

	r := strings.NewReader(config)
	cfg, err := Load(r)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, cfg)
}
