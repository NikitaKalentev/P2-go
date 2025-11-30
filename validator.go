package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	snakeCaseRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*[a-z0-9]$`)
	memRegex       = regexp.MustCompile(`^\d+(Gi|Mi|Ki)$`)
)

type ValidationError struct {
	File string
	Line int
	Msg  string
}

func (e *ValidationError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d %s", e.File, e.Line, e.Msg)
	}
	return fmt.Sprintf("%s %s", e.File, e.Msg)
}

func NewValidationError(file string, line int, msg string) error {
	return &ValidationError{File: file, Line: line, Msg: msg}
}

// ValidatePod выполняет полную валидацию Pod из YAML-дерева
func ValidatePod(filepath string, root *yaml.Node) error {
	if len(root.Content) == 0 {
		return NewValidationError(filepath, 0, "empty YAML content")
	}

	doc := root.Content[0] // первый документ
	if doc.Kind != yaml.MappingNode {
		return NewValidationError(filepath, doc.Line, "root must be a mapping")
	}

	// Извлекаем поля верхнего уровня
	fields := map[string]*yaml.Node{}
	for i := 0; i < len(doc.Content); i += 2 {
		keyNode := doc.Content[i]
		valNode := doc.Content[i+1]
		if i+1 >= len(doc.Content) {
			return NewValidationError(filepath, keyNode.Line, "malformed mapping: missing value for key")
		}
		fields[keyNode.Value] = valNode
	}

	// apiVersion (обязательно, string, == "v1")
	if node, ok := fields["apiVersion"]; !ok {
		return NewValidationError(filepath, 0, "apiVersion is required")
	} else if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return NewValidationError(filepath, node.Line, "apiVersion must be string")
	} else if node.Value != "v1" {
		return NewValidationError(filepath, node.Line, fmt.Sprintf("apiVersion has unsupported value '%s'", node.Value))
	}

	// kind (обязательно, string, == "Pod")
	if node, ok := fields["kind"]; !ok {
		return NewValidationError(filepath, 0, "kind is required")
	} else if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return NewValidationError(filepath, node.Line, "kind must be string")
	} else if node.Value != "Pod" {
		return NewValidationError(filepath, node.Line, fmt.Sprintf("kind has unsupported value '%s'", node.Value))
	}

	// metadata (обязательно, ObjectMeta)
	if node, ok := fields["metadata"]; !ok {
		return NewValidationError(filepath, 0, "metadata is required")
	} else if err := validateObjectMeta(filepath, node); err != nil {
		return err
	}

	// spec (обязательно, PodSpec)
	if node, ok := fields["spec"]; !ok {
		return NewValidationError(filepath, 0, "spec is required")
	} else if err := validatePodSpec(filepath, node); err != nil {
		return err
	}

	return nil
}

func validateObjectMeta(filepath string, node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return NewValidationError(filepath, node.Line, "metadata must be a mapping")
	}

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		if i+1 >= len(node.Content) {
			return NewValidationError(filepath, keyNode.Line, "malformed metadata mapping")
		}
		fields[keyNode.Value] = valNode
	}

	// name (обязательно, string)
	if nameNode, ok := fields["name"]; !ok {
		return NewValidationError(filepath, 0, "metadata.name is required")
	} else if nameNode.Kind != yaml.ScalarNode || nameNode.Tag != "!!str" {
		return NewValidationError(filepath, nameNode.Line, "metadata.name must be string")
	}

	// namespace (необязательно, string)
	if nsNode, ok := fields["namespace"]; ok {
		if nsNode.Kind != yaml.ScalarNode || nsNode.Tag != "!!str" {
			return NewValidationError(filepath, nsNode.Line, "metadata.namespace must be string")
		}
	}

	// labels (необязательно, object)
	if labelsNode, ok := fields["labels"]; ok {
		if labelsNode.Kind != yaml.MappingNode {
			return NewValidationError(filepath, labelsNode.Line, "metadata.labels must be a mapping")
		}
		for i := 0; i < len(labelsNode.Content); i += 2 {
			if i+1 >= len(labelsNode.Content) {
				return NewValidationError(filepath, labelsNode.Line, "malformed metadata.labels mapping")
			}
			valNode := labelsNode.Content[i+1]
			if valNode.Kind != yaml.ScalarNode || valNode.Tag != "!!str" {
				return NewValidationError(filepath, valNode.Line, "metadata.labels value must be string")
			}
		}
	}

	return nil
}

func validatePodSpec(filepath string, node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return NewValidationError(filepath, node.Line, "spec must be a mapping")
	}

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		if i+1 >= len(node.Content) {
			return NewValidationError(filepath, keyNode.Line, "malformed spec mapping")
		}
		fields[keyNode.Value] = valNode
	}

	// os (необязательно, PodOS)
	if osNode, ok := fields["os"]; ok {
		if err := validatePodOS(filepath, osNode); err != nil {
			return err
		}
	}

	// containers (обязательно, []Container)
	if containersNode, ok := fields["containers"]; !ok {
		return NewValidationError(filepath, 0, "spec.containers is required")
	} else if containersNode.Kind != yaml.SequenceNode {
		return NewValidationError(filepath, containersNode.Line, "spec.containers must be a sequence")
	} else if len(containersNode.Content) == 0 {
		return NewValidationError(filepath, containersNode.Line, "spec.containers must not be empty")
	} else {
		for idx, containerNode := range containersNode.Content {
			if err := validateContainer(filepath, containerNode, idx+1); err != nil {
				return err
			}
		}
	}

	return nil
}

func validatePodOS(filepath string, node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return NewValidationError(filepath, node.Line, "spec.os must be a mapping")
	}

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		if i+1 >= len(node.Content) {
			return NewValidationError(filepath, keyNode.Line, "malformed spec.os mapping")
		}
		fields[keyNode.Value] = valNode
	}

	// name (обязательно, string, linux|windows)
	if nameNode, ok := fields["name"]; !ok {
		return NewValidationError(filepath, 0, "spec.os.name is required")
	} else if nameNode.Kind != yaml.ScalarNode || nameNode.Tag != "!!str" {
		return NewValidationError(filepath, nameNode.Line, "spec.os.name must be string")
	} else if nameNode.Value != "linux" && nameNode.Value != "windows" {
		return NewValidationError(filepath, nameNode.Line, fmt.Sprintf("spec.os.name has unsupported value '%s'", nameNode.Value))
	}

	return nil
}

func validateContainer(filepath string, node *yaml.Node, idx int) error {
	if node.Kind != yaml.MappingNode {
		return NewValidationError(filepath, node.Line, fmt.Sprintf("spec.containers[%d] must be a mapping", idx))
	}

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		if i+1 >= len(node.Content) {
			return NewValidationError(filepath, keyNode.Line, fmt.Sprintf("malformed spec.containers[%d] mapping", idx))
		}
		fields[keyNode.Value] = valNode
	}

	// name (обязательно, string, snake_case)
	if nameNode, ok := fields["name"]; !ok {
		return NewValidationError(filepath, 0, fmt.Sprintf("spec.containers[%d].name is required", idx))
	} else if nameNode.Kind != yaml.ScalarNode || nameNode.Tag != "!!str" {
		return NewValidationError(filepath, nameNode.Line, fmt.Sprintf("spec.containers[%d].name must be string", idx))
	} else if !snakeCaseRegex.MatchString(nameNode.Value) {
		return NewValidationError(filepath, nameNode.Line, fmt.Sprintf("spec.containers[%d].name has invalid format '%s'", idx, nameNode.Value))
	}

	// image (обязательно, string, registry.bigbrother.io/..., tag required)
	if imgNode, ok := fields["image"]; !ok {
		return NewValidationError(filepath, 0, fmt.Sprintf("spec.containers[%d].image is required", idx))
	} else if imgNode.Kind != yaml.ScalarNode || imgNode.Tag != "!!str" {
		return NewValidationError(filepath, imgNode.Line, fmt.Sprintf("spec.containers[%d].image must be string", idx))
	} else if !strings.HasPrefix(imgNode.Value, "registry.bigbrother.io/") {
		return NewValidationError(filepath, imgNode.Line, fmt.Sprintf("spec.containers[%d].image has invalid format '%s'", idx, imgNode.Value))
	} else {
		rest := imgNode.Value[len("registry.bigbrother.io/"):]
		parts := strings.Split(rest, ":")
		if len(parts) < 2 {
			return NewValidationError(filepath, imgNode.Line, fmt.Sprintf("spec.containers[%d].image has invalid format '%s' (missing tag)", idx, imgNode.Value))
		}
		tag := parts[len(parts)-1]
		if tag == "" {
			return NewValidationError(filepath, imgNode.Line, fmt.Sprintf("spec.containers[%d].image has invalid format '%s' (empty tag)", idx, imgNode.Value))
		}
	}

	// ports (необязательно, []ContainerPort)
	if portsNode, ok := fields["ports"]; ok {
		if portsNode.Kind != yaml.SequenceNode {
			return NewValidationError(filepath, portsNode.Line, fmt.Sprintf("spec.containers[%d].ports must be a sequence", idx))
		}
		for pi, portNode := range portsNode.Content {
			if err := validateContainerPort(filepath, portNode, idx, pi+1); err != nil {
				return err
			}
		}
	}

	// readinessProbe (необязательно, Probe)
	if probeNode, ok := fields["readinessProbe"]; ok {
		if err := validateProbe(filepath, probeNode, fmt.Sprintf("spec.containers[%d].readinessProbe", idx)); err != nil {
			return err
		}
	}

	// livenessProbe (необязательно, Probe)
	if probeNode, ok := fields["livenessProbe"]; ok {
		if err := validateProbe(filepath, probeNode, fmt.Sprintf("spec.containers[%d].livenessProbe", idx)); err != nil {
			return err
		}
	}

	// resources (обязательно, ResourceRequirements)
	if resNode, ok := fields["resources"]; !ok {
		return NewValidationError(filepath, 0, fmt.Sprintf("spec.containers[%d].resources is required", idx))
	} else if err := validateResourceRequirements(filepath, resNode, idx); err != nil {
		return err
	}

	return nil
}

func validateContainerPort(filepath string, node *yaml.Node, containerIdx, portIdx int) error {
	if node.Kind != yaml.MappingNode {
		return NewValidationError(filepath, node.Line, fmt.Sprintf("spec.containers[%d].ports[%d] must be a mapping", containerIdx, portIdx))
	}

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		if i+1 >= len(node.Content) {
			return NewValidationError(filepath, keyNode.Line, fmt.Sprintf("malformed spec.containers[%d].ports[%d] mapping", containerIdx, portIdx))
		}
		fields[keyNode.Value] = valNode
	}

	// containerPort (обязательно, int, 0 < x < 65536)
	if cpNode, ok := fields["containerPort"]; !ok {
		return NewValidationError(filepath, 0, fmt.Sprintf("spec.containers[%d].ports[%d].containerPort is required", containerIdx, portIdx))
	} else if cpNode.Kind != yaml.ScalarNode {
		return NewValidationError(filepath, cpNode.Line, fmt.Sprintf("spec.containers[%d].ports[%d].containerPort must be int", containerIdx, portIdx))
	} else {
		val, err := strconv.Atoi(cpNode.Value)
		if err != nil {
			return NewValidationError(filepath, cpNode.Line, fmt.Sprintf("spec.containers[%d].ports[%d].containerPort must be int", containerIdx, portIdx))
		}
		if val <= 0 || val >= 65536 {
			return NewValidationError(filepath, cpNode.Line, fmt.Sprintf("spec.containers[%d].ports[%d].containerPort value out of range", containerIdx, portIdx))
		}
	}

	// protocol (необязательно, string, TCP|UDP)
	if protoNode, ok := fields["protocol"]; ok {
		if protoNode.Kind != yaml.ScalarNode || protoNode.Tag != "!!str" {
			return NewValidationError(filepath, protoNode.Line, fmt.Sprintf("spec.containers[%d].ports[%d].protocol must be string", containerIdx, portIdx))
		}
		if protoNode.Value != "TCP" && protoNode.Value != "UDP" {
			return NewValidationError(filepath, protoNode.Line, fmt.Sprintf("spec.containers[%d].ports[%d].protocol has unsupported value '%s'", containerIdx, portIdx, protoNode.Value))
		}
	}

	return nil
}

func validateProbe(filepath string, node *yaml.Node, fieldPath string) error {
	if node.Kind != yaml.MappingNode {
		return NewValidationError(filepath, node.Line, fmt.Sprintf("%s must be a mapping", fieldPath))
	}

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		if i+1 >= len(node.Content) {
			return NewValidationError(filepath, keyNode.Line, fmt.Sprintf("malformed %s mapping", fieldPath))
		}
		fields[keyNode.Value] = valNode
	}

	// httpGet (обязательно, HTTPGetAction)
	if httpNode, ok := fields["httpGet"]; !ok {
		return NewValidationError(filepath, 0, fmt.Sprintf("%s.httpGet is required", fieldPath))
	} else if err := validateHTTPGetAction(filepath, httpNode, fieldPath+".httpGet"); err != nil {
		return err
	}

	return nil
}

func validateHTTPGetAction(filepath string, node *yaml.Node, fieldPath string) error {
	if node.Kind != yaml.MappingNode {
		return NewValidationError(filepath, node.Line, fmt.Sprintf("%s must be a mapping", fieldPath))
	}

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		if i+1 >= len(node.Content) {
			return NewValidationError(filepath, keyNode.Line, fmt.Sprintf("malformed %s mapping", fieldPath))
		}
		fields[keyNode.Value] = valNode
	}

	// path (обязательно, string, must be absolute)
	if pathNode, ok := fields["path"]; !ok {
		return NewValidationError(filepath, 0, fmt.Sprintf("%s.path is required", fieldPath))
	} else if pathNode.Kind != yaml.ScalarNode || pathNode.Tag != "!!str" {
		return NewValidationError(filepath, pathNode.Line, fmt.Sprintf("%s.path must be string", fieldPath))
	} else if !strings.HasPrefix(pathNode.Value, "/") {
		return NewValidationError(filepath, pathNode.Line, fmt.Sprintf("%s.path has invalid format '%s'", fieldPath, pathNode.Value))
	}

	// port (обязательно, int, 0 < x < 65536)
	if portNode, ok := fields["port"]; !ok {
		return NewValidationError(filepath, 0, fmt.Sprintf("%s.port is required", fieldPath))
	} else if portNode.Kind != yaml.ScalarNode {
		return NewValidationError(filepath, portNode.Line, fmt.Sprintf("%s.port must be int", fieldPath))
	} else {
		val, err := strconv.Atoi(portNode.Value)
		if err != nil {
			return NewValidationError(filepath, portNode.Line, fmt.Sprintf("%s.port must be int", fieldPath))
		}
		if val <= 0 || val >= 65536 {
			return NewValidationError(filepath, portNode.Line, fmt.Sprintf("%s.port value out of range", fieldPath))
		}
	}

	return nil
}

func validateResourceRequirements(filepath string, node *yaml.Node, containerIdx int) error {
	if node.Kind != yaml.MappingNode {
		return NewValidationError(filepath, node.Line, fmt.Sprintf("spec.containers[%d].resources must be a mapping", containerIdx))
	}

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		if i+1 >= len(node.Content) {
			return NewValidationError(filepath, keyNode.Line, fmt.Sprintf("malformed spec.containers[%d].resources mapping", containerIdx))
		}
		fields[keyNode.Value] = valNode
	}

	checkResource := func(field string, resourceNode *yaml.Node) error {
		if resourceNode == nil {
			return nil
		}
		if resourceNode.Kind != yaml.MappingNode {
			return NewValidationError(filepath, resourceNode.Line, fmt.Sprintf("spec.containers[%d].resources.%s must be a mapping", containerIdx, field))
		}

		for i := 0; i < len(resourceNode.Content); i += 2 {
			if i+1 >= len(resourceNode.Content) {
				return NewValidationError(filepath, resourceNode.Line, fmt.Sprintf("malformed spec.containers[%d].resources.%s mapping", containerIdx, field))
			}
			keyNode := resourceNode.Content[i]
			valNode := resourceNode.Content[i+1]
			switch keyNode.Value {
			case "cpu":
				if valNode.Kind != yaml.ScalarNode {
					return NewValidationError(filepath, valNode.Line, fmt.Sprintf("spec.containers[%d].resources.%s.cpu must be integer", containerIdx, field))
				}
				if _, err := strconv.Atoi(valNode.Value); err != nil {
					return NewValidationError(filepath, valNode.Line, fmt.Sprintf("spec.containers[%d].resources.%s.cpu must be integer", containerIdx, field))
				}
			case "memory":
				if valNode.Kind != yaml.ScalarNode || valNode.Tag != "!!str" {
					return NewValidationError(filepath, valNode.Line, fmt.Sprintf("spec.containers[%d].resources.%s.memory must be string", containerIdx, field))
				}
				if !memRegex.MatchString(valNode.Value) {
					return NewValidationError(filepath, valNode.Line, fmt.Sprintf("spec.containers[%d].resources.%s.memory has invalid format '%s'", containerIdx, field, valNode.Value))
				}
			default:
				return NewValidationError(filepath, keyNode.Line, fmt.Sprintf("spec.containers[%d].resources.%s has unsupported key '%s'", containerIdx, field, keyNode.Value))
			}
		}
		return nil
	}

	if err := checkResource("requests", fields["requests"]); err != nil {
		return err
	}
	if err := checkResource("limits", fields["limits"]); err != nil {
		return err
	}

	return nil
}