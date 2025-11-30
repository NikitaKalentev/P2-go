package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type PodAPIVersion struct {
	structure.Element
}

func (n *PodAPIVersion) Validate() {
	n.RequireType("str")
	if n.GetValue().Value != "v1" {
		n.ErrorUnsupportedValue()
	}
}

func NewPodAPIVersion() *PodAPIVersion {
	return &PodAPIVersion{}
}