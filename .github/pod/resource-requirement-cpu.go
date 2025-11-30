package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type ResourceRequirementCPU struct {
	structure.Element
}

func (n *ResourceRequirementCPU) Validate() {
	n.RequireType("int")
}

func NewResourceRequirementCPU() *ResourceRequirementCPU {
	return &ResourceRequirementCPU{}
}