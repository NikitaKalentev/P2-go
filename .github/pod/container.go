package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type Container struct {
	structure.ContainerMap
}

func NewContainer() *Container {
	n := Container{}
	n.AddElement("name", NewContainerName())
	n.AddElement("image", NewContainerImage())
	n.AddElement("ports", NewContainerPorts())
	n.AddElement("readinessProbe", NewProbe())
	n.AddElement("livenessProbe", NewProbe())
	n.AddElement("resources", NewResourceRequirements())

	return &n
}

func (n *Container) Validate() {
	n.RequireChildNode("name")
	n.RequireChildNode("image")
	n.RequireChildNode("resources")

	n.ContainerMap.Validate()
}