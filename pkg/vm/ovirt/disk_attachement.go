package ovirt

import "encoding/xml"

type DiskAttachments struct {
	Attachments []DiskAttachment
}

type DiskAttachment struct {
	ID       string `xml:"id,attr"`
	Bootable bool   `xml:"bootable"`
}

type NewDiskAttachment struct {
	XMLName     struct{} `xml:"disk_attachment"`
	Bootable    bool     `xml:"bootable"`
	PassDiscard bool     `xml:"pass_discard"`
	Interface   string   `xml:"interface"`
	Active      bool     `xml:"active"`
	Disk        Disk     `xml:"disk"`
}

func (nd *NewDiskAttachment) serialize() []byte {
	b, _ := xml.Marshal(nd)
	return b
}
