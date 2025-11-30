package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type PodKind struct {
	structure.Element
}

func (n *PodKind) Validate() {
	n.RequireType("str")
	if n.GetValue().Value != "Pod" {
		n.ErrorUnsupportedValue()
	}
}

func NewPodKind() *PodKind {
	return &PodKind{}
}