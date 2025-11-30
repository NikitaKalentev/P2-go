package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type HTTPGetAction struct {
	structure.ContainerMap
}

func NewHTTPGetAction() *HTTPGetAction {
	n := HTTPGetAction{}
	n.AddElement("path", NewAbsolutePath())
	n.AddElement("port", NewPort())

	return &n
}

func (n *HTTPGetAction) Validate() {
	n.RequireChildNode("path")
	n.RequireChildNode("port")

	n.ContainerMap.Validate()
}