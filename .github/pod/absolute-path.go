package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
)

type AbsolutePath struct {
	structure.Element
}

func (n *AbsolutePath) Validate() {
	n.RequireType("str")
	if value := n.GetValue().Value; len(value) > 0 {
		if string(value[0]) != "/" {
			n.ErrorUnsupportedValue()
		}
	} else {
		n.RequireValue()
	}
}

func NewAbsolutePath() *AbsolutePath {
	return &AbsolutePath{}
}