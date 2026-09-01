package main

import (
	"testing"

	"github.com/MauveSoftware/provisionize/cmd/provisionize/config"
	"github.com/MauveSoftware/provisionize/pkg/api/proto"
	"github.com/stretchr/testify/assert"
)

func TestTowerTemplateIDsForVM(t *testing.T) {
	tests := []struct {
		name     string
		vm       *proto.VirtualMachine
		expected []uint
	}{
		{
			name:     "known template",
			vm:       &proto.VirtualMachine{Template: "linux"},
			expected: []uint{1, 2},
		},
		{
			name:     "unknown template",
			vm:       &proto.VirtualMachine{Template: "does-not-exist"},
			expected: []uint{},
		},
	}

	t.Parallel()

	m := newTemplateManager([]*config.ProvisionTemplate{
		{Name: "linux", AnsibleTemplates: []uint{1, 2}},
		{Name: "windows", AnsibleTemplates: []uint{3}},
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ids := m.TowerTemplateIDsForVM(test.vm)

			assert.Equal(t, test.expected, ids, "template IDs for %s did not match expectation", test.name)
		})
	}
}
