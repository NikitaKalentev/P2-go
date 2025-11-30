package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type Pod struct {
	structure.ContainerMap
}

func NewPod() *Pod {
	n := Pod{}
	n.AddElement("apiVersion", NewPodAPIVersion())
	n.AddElement("kind", NewPodKind())
	n.AddElement("metadata", NewObjectMeta())
	n.AddElement("spec", NewPodSpec())

	return &n
}

func (n *Pod) Validate() {
	n.RequireChildNode("apiVersion")
	n.RequireChildNode("kind")
	n.RequireChildNode("metadata")
	n.RequireChildNode("spec")

	n.ContainerMap.Validate()
}