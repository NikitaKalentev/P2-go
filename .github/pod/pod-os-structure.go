package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type PodOsStructure struct {
	structure.ContainerMap
}

func NewPodOsStructure() *PodOsStructure {
	n := PodOsStructure{}
	n.AddElement("name", NewPodOsName())

	return &n
}

func (n *PodOsStructure) Validate() {
	n.RequireChildNode("name")

	n.ContainerMap.Validate()
}