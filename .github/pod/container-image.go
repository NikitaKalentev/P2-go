package pod

import (
	"strings"

	"github.com/NikitaKalentev/P2-go/structure"
)

type ContainerImage struct {
	structure.Element
}

func (n *ContainerImage) Validate() {
	n.RequireType("str")
	parts := strings.Split(n.GetValue().Value, "/")
	if len(parts) < 2 || parts[0] != "registry.bigbrother.io" {
		n.ErrorUnsupportedValue()
	} else {
		parts = strings.Split(parts[len(parts)-1], ":")
		if len(parts) != 2 {
			n.ErrorUnsupportedValue()
		}
	}
}

func NewContainerImage() *ContainerImage {
	return &ContainerImage{}
}