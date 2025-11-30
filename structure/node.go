package structure

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type NodeValuer interface {
	SetValue(node *yaml.Node)
	GetValue() *yaml.Node
}

type Node struct {
	Value *yaml.Node
}

func (n *Node) SetValue(node *yaml.Node) {
	n.Value = node
}

func (n Node) GetValue() *yaml.Node {
	return n.Value
}

func (n Node) DebugValue() {
	fmt.Printf("%#v\n\n", n.Value)
}