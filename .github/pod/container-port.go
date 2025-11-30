package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type ContainerPort struct {
	structure.ContainerMap
}

func NewContainerPort() *ContainerPort {
	n := ContainerPort{}
	n.AddElement("containerPort", NewPort())
	n.AddElement("protocol", NewContainerPortProtocol())

	return &n
}

func (n *ContainerPort) Validate() {
	n.RequireChildNode("containerPort")

	n.ContainerMap.Validate()
}