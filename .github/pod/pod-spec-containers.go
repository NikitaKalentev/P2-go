package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
	"gopkg.in/yaml.v3"
)

type SpecContainers struct {
	structure.ContainerList
}

func NewSpecContainers() *SpecContainers {
	n := SpecContainers{}
	n.Create = func(value *yaml.Node) structure.NodeReporter {
		return NewContainer()
	}

	return &n
}

func (n *SpecContainers) Validate() {
	n.ContainerList.Validate()
}