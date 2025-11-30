package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type ResourceRequirements struct {
	structure.ContainerMap
}

func NewResourceRequirements() *ResourceRequirements {
	n := ResourceRequirements{}
	n.AddElement("requests", NewResourceRequirement())
	n.AddElement("limits", NewResourceRequirement())

	return &n
}

func (n *ResourceRequirements) Validate() {
	n.ContainerMap.Validate()
}