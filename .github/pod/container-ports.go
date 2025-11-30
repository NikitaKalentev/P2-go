package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
	"gopkg.in/yaml.v3"
)

type ContainerPorts struct {
	structure.ContainerList
}

func NewContainerPorts() *ContainerPorts {
	n := ContainerPorts{}
	n.Create = func(value *yaml.Node) structure.NodeReporter {
		return NewContainerPort()
	}

	return &n
}

func (n *ContainerPorts) Validate() {
	n.ContainerList.Validate()
}