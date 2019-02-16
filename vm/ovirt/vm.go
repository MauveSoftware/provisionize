package ovirt

// VM represents an oVirt VM
type VM struct {
	ID     string `xml:"id,attr"`
	Name   string `xml:"name"`
	Status string `xml:"status"`
}
