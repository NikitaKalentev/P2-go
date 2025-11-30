package structure

import "gopkg.in/yaml.v3"

type VariantCreate func(name *yaml.Node, value *yaml.Node) map[string][]NodeReporter

type Variants struct {
	Element
	Alternatives []NodeReporter
	Create       VariantCreate
}

func (n *Variants) Accept(name *yaml.Node, value *yaml.Node) {
	n.Element.Accept(name, value)
	if n.Create != nil {
		alternatives, ok := n.Create(name, value)[value.Tag]
		if ok {
			n.Alternatives = alternatives
		}
		for _, node := range n.Alternatives {
			node.Accept(name, value)
		}
	}
}

func (n *Variants) Validate() {
	if len(n.Alternatives) == 0 {
		n.ErrorUnsupportedValue()
	}
	for _, node := range n.Alternatives {
		node.Validate()
	}
}

func (n Variants) GetErrors() []string {
	var errors []string

	copy(errors, n.Errors)
	if len(errors) == 0 {
		for _, node := range n.Alternatives {
			errors = node.GetErrors()
			if len(errors) == 0 {
				break
			}
		}
	}

	return errors
}