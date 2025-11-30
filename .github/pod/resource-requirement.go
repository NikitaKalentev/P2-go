package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type ResourceRequirement struct {
	structure.ContainerMap
}

func NewResourceRequirement() *ResourceRequirement {
	n := ResourceRequirement{}
	n.AddElement("cpu", NewResourceRequirementCPU())
	n.AddElement("memory", NewResourceRequirementMemory())

	return &n
}

func (n *ResourceRequirement) Validate() {
	n.ContainerMap.Validate()
}