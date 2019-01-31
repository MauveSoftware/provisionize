package vm

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MauveSoftware/provisionize/api/proto"
)

const testTemplate = `<vm>
	<name>{{.Name}}</name>
 	<cluster>
		<name>{{.ClusterName}}</name>
	</cluster>
	<template>
		<name>{{.TemplateName}}</name>
	</template>
	<disks>
		<clone>true</clone>
	</disks>
	<memory>{{mb_to_byte .MemoryMb}}</memory>
	<cpu>
		<topology>
			<cores>1</cores>
			<sockets>{{.CpuCores}}</sockets>
		</topology>
	</cpu>
	<initialization>
		<cloud_init>
			<host>
				<address>{{.Name}}</address>
			</host>
			<network_configuration>
				<nics>
					<nic>
						<name>ens3</name>
						<boot_protocol>static</boot_protocol>
						<network>
							<ip address="{{.Ipv4.Address}}" netmask="{{.Ipv4.PrefixLength}}" gateway="{{.Ipv4.Gateway}}" />
							<ip address="{{.Ipv6.Address}}" netmask="{{.Ipv6.PrefixLength}}" gateway="{{.Ipv6.Gateway}}" />
						</network>
						<on_boot>true</on_boot>
					</nic>
				</nics>
			</network_configuration>
		</cloud_init>
	</initialization>
</vm>`

func TestGetVMCreateRequest(t *testing.T) {
	expected := `<vm>
	<name>testhost</name>
	<cluster>
	  <name>cluster1</name>
	</cluster>
	<template>
	  <name>template1</name>
	</template>
	<disks>
	  <clone>true</clone>
	</disks>
	<memory>1073741824</memory>
	<cpu>
	  <topology>
		<cores>1</cores>
		<sockets>4</sockets>
	  </topology>
	</cpu>
	<initialization>
	  <cloud_init>
		<host>
		  <address>testhost</address>
		</host>
		<network_configuration>
		  <nics>
			<nic>
			  <name>ens3</name>
			  <boot_protocol>static</boot_protocol>
			  <network>
				<ip address="192.168.1.100" netmask="32" gateway="192.168.1.1" />
				<ip address="2001:678:1e0::f00" netmask="128" gateway="2001:678:1e0::1" />
			  </network>
			  <on_boot>true</on_boot>
			</nic>
		  </nics>
		</network_configuration>
	  </cloud_init>
	</initialization>
  </vm>`

	vm := &proto.VirtualMachine{
		Name:         "testhost",
		ClusterName:  "cluster1",
		TemplateName: "template1",
		CpuCores:     4,
		Ipv4: &proto.IPConfig{
			Address:      "192.168.1.100",
			PrefixLength: 32,
			Gateway:      "192.168.1.1",
		},
		Ipv6: &proto.IPConfig{
			Address:      "2001:678:1e0::f00",
			PrefixLength: 128,
			Gateway:      "2001:678:1e0::1",
		},
		MemoryMb: 1024,
	}

	svc := &OvirtService{
		template: testTemplate,
	}
	r, err := svc.getVMCreateRequest(vm)
	if err != nil {
		t.Fatal(err)
	}

	b, _ := ioutil.ReadAll(r)
	assert.Equal(t, unify(expected), unify(string(b)))
}

func unify(str string) string {
	str = strings.Replace(str, "\t", "", -1)
	return strings.Replace(str, " ", "", -1)
}
