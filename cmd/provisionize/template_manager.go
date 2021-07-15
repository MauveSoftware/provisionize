package main

import (
	"github.com/MauveSoftware/provisionize/cmd/provisionize/config"
	"github.com/MauveSoftware/provisionize/pkg/api/proto"
)

type templateManager struct {
	templates map[string]*config.ProvisionTemplate
}

func newTemplateManager(templates []*config.ProvisionTemplate) *templateManager {
	m := make(map[string]*config.ProvisionTemplate)
	for _, t := range templates {
		m[t.Name] = t
	}

	return &templateManager{templates: m}
}

func (t *templateManager) OvirtTemplateNameForVM(vm *proto.VirtualMachine) string {
	if template, found := t.templates[vm.Template]; found {
		return template.OvirtTemplate
	}

	return ""
}

func (t *templateManager) TowerTemplateIDsForVM(vm *proto.VirtualMachine) []uint {
	if template, found := t.templates[vm.Template]; found {
		return template.AnsibleTemplates
	}

	return []uint{}
}
