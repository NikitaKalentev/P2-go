package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type ObjectMetaNamespace struct {
	structure.Element
}

func (n *ObjectMetaNamespace) Validate() {
	n.RequireType("str")
	n.RequireValue()
}

func NewObjectMetaNamespace() *ObjectMetaNamespace {
	return &ObjectMetaNamespace{}
}