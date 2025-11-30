package structure

import (
	"gopkg.in/yaml.v3"
)

type Container interface {
	GetChildNodes() []NodeReporter
}

type NodeReporter interface {
	Accept(name *yaml.Node, value *yaml.Node)
	Initialized() bool
	GetErrors() []string
	Validate()
}

type NodeReport struct {
	Errors []string
}

func (n NodeReport) GetErrors() []string {
	return n.Errors
}

func (n *NodeReport) AddError(err string) {
	n.Errors = append(n.Errors, err)
}