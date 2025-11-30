package structure

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type ContainerMap struct {
	Element
	ChildNodes map[string]NodeReporter
}

func (n *ContainerMap) Accept(name *yaml.Node, value *yaml.Node) {
	n.Element.Accept(name, value)

	content := value.Content
	for i := 0; i < len(content); i = i + 2 {
		name := content[i]
		childNode, exists := n.ChildNodes[name.Value]
		if exists {
			childNode.Accept(name, content[i+1])
		}
	}
}

func (n *ContainerMap) Validate() {
	if n.RequireType("map") {
		for _, node := range n.GetChildNodes() {
			if node.Initialized() {
				node.Validate()
			}
		}
	}
}

func (n ContainerMap) GetErrors() []string {
	errors := append(make([]string, 0, 16), n.Errors...)
	for _, node := range n.GetChildNodes() {
		errors = append(errors, node.GetErrors()...)
	}
	return errors
}

func (n *ContainerMap) GetChildNodes() []NodeReporter {
	n.initializeMap()
	nodes := make([]NodeReporter, 0, len(n.ChildNodes))
	for _, node := range n.ChildNodes {
		nodes = append(nodes, node)
	}

	return nodes
}

func (n *ContainerMap) AddElement(key string, node NodeReporter) {
	n.initializeMap()
	n.ChildNodes[key] = node
}

func (n *ContainerMap) GetElement(key string) NodeReporter {
	n.initializeMap()
	if node, ok := n.ChildNodes[key]; ok {
		return node
	}
	return nil
}

func (n *ContainerMap) RequireChildNode(key string) bool {
	node := n.GetElement(key)
	if node == nil || !node.Initialized() {
		n.AddError(fmt.Sprintf("%s is required", key))
		return false
	}
	return true
}

func (n *ContainerMap) initializeMap() {
	if n.ChildNodes == nil {
		n.ChildNodes = make(map[string]NodeReporter)
	}
}