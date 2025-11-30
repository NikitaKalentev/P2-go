package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type ObjectMeta struct {
	structure.ContainerMap
}

func NewObjectMeta() *ObjectMeta {
	n := ObjectMeta{}
	n.AddElement("name", NewObjectMetaName())
	n.AddElement("namespace", NewObjectMetaNamespace())
	n.AddElement("labels", NewObjectMetaLabels())

	return &n
}

func (n *ObjectMeta) Validate() {
	n.RequireChildNode("name")

	n.ContainerMap.Validate()
}