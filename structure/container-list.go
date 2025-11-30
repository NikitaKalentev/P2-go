package structure

import (
	"gopkg.in/yaml.v3"
)

type ContainerList struct {
	Element
	ChildNodes []NodeReporter
	Create     func(value *yaml.Node) NodeReporter
}

func (n *ContainerList) Accept(name *yaml.Node, value *yaml.Node) {
	n.Element.Accept(name, value)
	if n.Create != nil {
		for _, item := range value.Content {
			node := n.Create(item)
			node.Accept(nil, item)
			n.AddElement(node)
		}
	}
}

func (n *ContainerList) Validate() {
	if n.RequireType("seq") {
		for _, node := range n.GetChildNodes() {
			node.Validate()
		}
	}
}

func (n ContainerList) GetErrors() []string {
	errors := append(make([]string, 0, 16), n.Errors...)
	for _, node := range n.GetChildNodes() {
		errors = append(errors, node.GetErrors()...)
	}
	return errors
}

func (n *ContainerList) GetChildNodes() []NodeReporter {
	return n.ChildNodes
}

func (n *ContainerList) AddElement(node NodeReporter) {
	n.ChildNodes = append(n.ChildNodes, node)
}