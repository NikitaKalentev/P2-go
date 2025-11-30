package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type Probe struct {
	structure.ContainerMap
}

func NewProbe() *Probe {
	n := Probe{}
	n.AddElement("httpGet", NewHTTPGetAction())

	return &n
}

func (n *Probe) Validate() {
	n.RequireChildNode("httpGet")

	n.ContainerMap.Validate()
}