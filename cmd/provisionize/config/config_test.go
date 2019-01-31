package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	config := `listen_address: "[::]:1337"
ovirt:
  url: https://my-ovirt.instance
  username: provisionize
  password: allTheThings
  template_path: /etc/provisionize/template
gcloud:
  project_id: "123456"`
	expected := &Config{
		ListenAddress: "[::]:1337",
		Ovirt: &OvirtConfig{
			Username:     "provisionize",
			Password:     "allTheThings",
			TemplatePath: "/etc/provisionize/template",
			URL:          "https://my-ovirt.instance",
		},
		GooglecCloudDNS: &GoogleCloudDNSConfig{
			ProjectID: "123456",
		},
	}

	r := strings.NewReader(config)
	cfg, err := Load(r)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, cfg)
}
