package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type ObjectMetaLabels struct {
	structure.ContainerMap
}

func (n *ObjectMetaLabels) Validate() {
	// @todo строковые ключ-значение?
}

func NewObjectMetaLabels() *ObjectMetaLabels {
	return &ObjectMetaLabels{}
}