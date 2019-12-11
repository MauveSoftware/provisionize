package ovirt

// VMs represents a list of VMs
type VMs struct {
	VMs []VM `xml:"vm"`
}

// VM represents an oVirt VM
type VM struct {
	ID     string `xml:"id,attr"`
	Name   string `xml:"name"`
	Status string `xml:"status"`
}
