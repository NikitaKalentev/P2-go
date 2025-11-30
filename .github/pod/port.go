package pod

import (
	"strconv"

	"github.com/NikitaKalentev/P2-go/structure"
)

type Port struct {
	structure.Element
}

func (n *Port) Validate() {
	n.RequireType("int")
	if value, err := strconv.Atoi(n.GetValue().Value); err != nil || value <= 0 || value >= 65536 {
		n.ErrorOutOfRange()
	}
}

func NewPort() *Port {
	return &Port{}
}