package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type PodOsName struct {
	structure.Element
}

func (n *PodOsName) Validate() {
	n.RequireType("str")
	if value := n.GetValue().Value; value != "linux" && value != "windows" {
		n.ErrorUnsupportedValue()
	}
}

func NewPodOsName() *PodOsName {
	return &PodOsName{}
}