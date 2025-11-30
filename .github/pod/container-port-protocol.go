package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type ContainerPortProtocol struct {
	structure.Element
}

func (n *ContainerPortProtocol) Validate() {
	n.RequireType("str")
	if value := n.GetValue().Value; value != "TCP" && value != "UDP" {
		n.ErrorUnsupportedValue()
	}
}

func NewContainerPortProtocol() *ContainerPortProtocol {
	return &ContainerPortProtocol{}
}