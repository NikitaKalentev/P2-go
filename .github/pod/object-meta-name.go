package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type ObjectMetaName struct {
	structure.Element
}

func (n *ObjectMetaName) Validate() {
	n.RequireType("str")
	n.RequireValue()
}

func NewObjectMetaName() *ObjectMetaName {
	return &ObjectMetaName{}
}