package structure

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type NodeNamer interface {
	SetName(node *yaml.Node)
	GetName() *yaml.Node
}

type Element struct {
	Node
	NodeReport
	Name *yaml.Node
}

func (n *Element) Accept(name *yaml.Node, value *yaml.Node) {
	n.SetName(name)
	n.SetValue(value)
}

func (n *Element) Initialized() bool {
	return n.GetName() != nil || n.GetValue() != nil
}

func (n *Element) Validate() {
}

func (n *Element) SetName(node *yaml.Node) {
	n.Name = node
}

func (n Element) GetName() *yaml.Node {
	return n.Name
}

func (n *Element) AddError(err string) {
	err = fmt.Sprintf("%d %s", n.GetValue().Line, err)

	n.NodeReport.AddError(err)
}

func (n *Element) RequireType(name string) bool {
	if n.GetValue().Tag != fmt.Sprintf("!!%s", name) {
		n.AddError(fmt.Sprintf("%s must be %s", n.GetName().Value, name))
		return false
	}
	return true
}

func (n *Element) RequireValue() {
	if n.GetValue().Value == "" {
		n.AddError(fmt.Sprintf("%s is required", n.GetName().Value))
	}
}

func (n *Element) ErrorUnsupportedValue() {
	n.AddError(fmt.Sprintf("%s has unsupported value '%s'", n.GetName().Value, n.GetValue().Value))
}

func (n *Element) ErrorInvalidFormat() {
	n.AddError(fmt.Sprintf("%s has invalid format '%s'", n.GetName().Value, n.GetValue().Value))
}

func (n *Element) ErrorOutOfRange() {
	n.AddError(fmt.Sprintf("%s value out of range", n.GetName().Value))
}