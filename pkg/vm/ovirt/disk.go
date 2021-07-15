package ovirt

type Disks struct {
	Disks []Disk `xml:"disk"`
}

type Disk struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,omitempty"`
}
