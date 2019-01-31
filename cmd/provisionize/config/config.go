package config

// Config represents the configuration
type Config struct {
	APIListenAddress string
	Ovirt            *OvirtConfig
	GooglecCloudDNS  *GoogleCloudDNSConfig
}

// OvirtConfig represents to oVirt configuration part
type OvirtConfig struct {
	URL      string
	Username string
	Password string
}

// GoogleCloudDNSConfig represents to DNS configuration part
type GoogleCloudDNSConfig struct {
	ProjectID string
}
