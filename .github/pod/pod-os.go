package pod

import (
	"github.com/NikitaKalentev/P2-go/structure"
	"gopkg.in/yaml.v3"
)

type PodOs struct {
	structure.Variants
}

func NewPodOs() *PodOs {
	n := PodOs{}
	n.Create = func(name *yaml.Node, value *yaml.Node) map[string][]structure.NodeReporter {
		var alternatives = make(map[string][]structure.NodeReporter, 0)

		alternatives["!!str"] = []structure.NodeReporter{NewPodOsName()}
		alternatives["!!map"] = []structure.NodeReporter{NewPodOsStructure()}

		return alternatives
	}

	return &n
}