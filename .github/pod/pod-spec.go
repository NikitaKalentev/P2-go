package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type PodSpec struct {
	structure.ContainerMap
}

func NewPodSpec() *PodSpec {
	n := PodSpec{}
	n.AddElement("os", NewPodOs())
	n.AddElement("containers", NewSpecContainers())

	return &n
}

func (n *PodSpec) Validate() {
	n.RequireChildNode("containers")

	n.ContainerMap.Validate()
}