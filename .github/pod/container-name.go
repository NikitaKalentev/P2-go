package pod

import (
	"strings"

	"github.com/NikitaKalentev/P2-go/structure"
)

type ContainerName struct {
	structure.Element
}

func (n *ContainerName) Validate() {
	n.RequireType("str")
	n.RequireValue()
	if value := n.GetValue().Value; value != strings.ToLower(value) {
		n.ErrorInvalidFormat()
	}
}

func NewContainerName() *ContainerName {
	return &ContainerName{}
}