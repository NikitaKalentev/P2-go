package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type ValidationError struct {
	File  string
	Line  int
	Field string
	Error string
}

func (e ValidationError) String() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d %s", e.File, e.Line, e.Error)
	}
	return fmt.Sprintf("%s %s", e.File, e.Error)
}

type Validator struct {
	errors []ValidationError
	file   string
}

func (v *Validator) addError(line int, field, message string) {
	v.errors = append(v.errors, ValidationError{
		File:  v.file,
		Line:  line,
		Field: field,
		Error: fmt.Sprintf("%s %s", field, message),
	})
}

func (v *Validator) validateTopLevel(node *yaml.Node) bool {
	valid := true
	fields := make(map[string]*yaml.Node)

	// Собираем поля верхнего уровня
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}
		key := node.Content[i]
		value := node.Content[i+1]
		fields[key.Value] = value
	}

	// Проверяем обязательные поля
	if apiVersion, exists := fields["apiVersion"]; !exists {
		v.addError(node.Line, "apiVersion", "is required")
		valid = false
	} else if apiVersion.Value != "v1" {
		v.addError(apiVersion.Line, "apiVersion", "must be 'v1'")
		valid = false
	}

	if kind, exists := fields["kind"]; !exists {
		v.addError(node.Line, "kind", "is required")
		valid = false
	} else if kind.Value != "Pod" {
		v.addError(kind.Line, "kind", "must be 'Pod'")
		valid = false
	}

	if metadata, exists := fields["metadata"]; !exists {
		v.addError(node.Line, "metadata", "is required")
		valid = false
	} else {
		v.validateObjectMeta(metadata)
	}

	if spec, exists := fields["spec"]; !exists {
		v.addError(node.Line, "spec", "is required")
		valid = false
	} else {
		v.validatePodSpec(spec)
	}

	return valid
}

func (v *Validator) validateObjectMeta(node *yaml.Node) {
	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}
		key := node.Content[i]
		value := node.Content[i+1]
		fields[key.Value] = value
	}

	if name, exists := fields["name"]; !exists {
		v.addError(node.Line, "metadata.name", "is required")
	} else if name.Value == "" {
		v.addError(name.Line, "metadata.name", "must be non-empty string")
	}

	if namespace, exists := fields["namespace"]; exists && namespace.Kind != yaml.ScalarNode {
		v.addError(namespace.Line, "metadata.namespace", "must be string")
	}

	if labels, exists := fields["labels"]; exists && labels.Kind != yaml.MappingNode {
		v.addError(labels.Line, "metadata.labels", "must be object")
	}
}

func (v *Validator) validatePodSpec(node *yaml.Node) {
	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}
		key := node.Content[i]
		value := node.Content[i+1]
		fields[key.Value] = value
	}

	if osNode, exists := fields["os"]; exists {
		v.validatePodOS(osNode)
	}

	if containers, exists := fields["containers"]; !exists {
		v.addError(node.Line, "spec.containers", "is required")
	} else {
		v.validateContainers(containers)
	}
}

func (v *Validator) validatePodOS(node *yaml.Node) {
	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}
		key := node.Content[i]
		value := node.Content[i+1]
		fields[key.Value] = value
	}

	if name, exists := fields["name"]; !exists {
		v.addError(node.Line, "spec.os.name", "is required")
	} else if name.Value != "linux" && name.Value != "windows" {
		v.addError(name.Line, "spec.os.name", fmt.Sprintf("has unsupported value '%s'", name.Value))
	}
}

func (v *Validator) validateContainers(node *yaml.Node) {
	if node.Kind != yaml.SequenceNode {
		v.addError(node.Line, "spec.containers", "must be array")
		return
	}

	containerNames := make(map[string]bool)
	snakeCaseRegex := regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)
	imageRegex := regexp.MustCompile(`^registry\.bigbrother\.io/[^:]+:[^:]+$`)

	for _, containerNode := range node.Content {
		fields := make(map[string]*yaml.Node)
		for i := 0; i < len(containerNode.Content); i += 2 {
			if i+1 >= len(containerNode.Content) {
				break
			}
			key := containerNode.Content[i]
			value := containerNode.Content[i+1]
			fields[key.Value] = value
		}

		// Проверка имени контейнера
		if name, exists := fields["name"]; !exists {
			v.addError(containerNode.Line, "spec.containers[].name", "is required")
		} else {
			if !snakeCaseRegex.MatchString(name.Value) {
				v.addError(name.Line, "spec.containers[].name", fmt.Sprintf("has invalid format '%s'", name.Value))
			}
			if containerNames[name.Value] {
				v.addError(name.Line, "spec.containers[].name", "must be unique within pod")
			}
			containerNames[name.Value] = true
		}

		// Проверка image
		if image, exists := fields["image"]; !exists {
			v.addError(containerNode.Line, "spec.containers[].image", "is required")
		} else if !imageRegex.MatchString(image.Value) {
			v.addError(image.Line, "spec.containers[].image", fmt.Sprintf("has invalid format '%s'", image.Value))
		}

		// Проверка ports
		if ports, exists := fields["ports"]; exists {
			v.validateContainerPorts(ports)
		}

		// Проверка readinessProbe
		if readinessProbe, exists := fields["readinessProbe"]; exists {
			v.validateProbe(readinessProbe, "readinessProbe")
		}

		// Проверка livenessProbe
		if livenessProbe, exists := fields["livenessProbe"]; exists {
			v.validateProbe(livenessProbe, "livenessProbe")
		}

		// Проверка resources
		if resources, exists := fields["resources"]; !exists {
			v.addError(containerNode.Line, "spec.containers[].resources", "is required")
		} else {
			v.validateResourceRequirements(resources)
		}
	}
}

func (v *Validator) validateContainerPorts(node *yaml.Node) {
	if node.Kind != yaml.SequenceNode {
		v.addError(node.Line, "spec.containers[].ports", "must be array")
		return
	}

	for _, portNode := range node.Content {
		fields := make(map[string]*yaml.Node)
		for i := 0; i < len(portNode.Content); i += 2 {
			if i+1 >= len(portNode.Content) {
				break
			}
			key := portNode.Content[i]
			value := portNode.Content[i+1]
			fields[key.Value] = value
		}

		if containerPort, exists := fields["containerPort"]; !exists {
			v.addError(portNode.Line, "spec.containers[].ports[].containerPort", "is required")
		} else {
			if containerPort.Kind != yaml.ScalarNode {
				v.addError(containerPort.Line, "spec.containers[].ports[].containerPort", "must be integer")
			} else {
				port, err := strconv.Atoi(containerPort.Value)
				if err != nil {
					v.addError(containerPort.Line, "spec.containers[].ports[].containerPort", "must be integer")
				} else if port <= 0 || port >= 65536 {
					v.addError(containerPort.Line, "spec.containers[].ports[].containerPort", "value out of range")
				}
			}
		}

		if protocol, exists := fields["protocol"]; exists {
			if protocol.Value != "TCP" && protocol.Value != "UDP" {
				v.addError(protocol.Line, "spec.containers[].ports[].protocol", fmt.Sprintf("has unsupported value '%s'", protocol.Value))
			}
		}
	}
}

func (v *Validator) validateProbe(node *yaml.Node, probeType string) {
	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}
		key := node.Content[i]
		value := node.Content[i+1]
		fields[key.Value] = value
	}

	if httpGet, exists := fields["httpGet"]; !exists {
		v.addError(node.Line, fmt.Sprintf("spec.containers[].%s.httpGet", probeType), "is required")
	} else {
		v.validateHTTPGetAction(httpGet, probeType)
	}
}

func (v *Validator) validateHTTPGetAction(node *yaml.Node, probeType string) {
	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}
		key := node.Content[i]
		value := node.Content[i+1]
		fields[key.Value] = value
	}

	if path, exists := fields["path"]; !exists {
		v.addError(node.Line, fmt.Sprintf("spec.containers[].%s.httpGet.path", probeType), "is required")
	} else if path.Value == "" || !strings.HasPrefix(path.Value, "/") {
		v.addError(path.Line, fmt.Sprintf("spec.containers[].%s.httpGet.path", probeType), "must be absolute path")
	}

	if port, exists := fields["port"]; !exists {
		v.addError(node.Line, fmt.Sprintf("spec.containers[].%s.httpGet.port", probeType), "is required")
	} else {
		if port.Kind != yaml.ScalarNode {
			v.addError(port.Line, fmt.Sprintf("spec.containers[].%s.httpGet.port", probeType), "must be integer")
		} else {
			portNum, err := strconv.Atoi(port.Value)
			if err != nil {
				v.addError(port.Line, fmt.Sprintf("spec.containers[].%s.httpGet.port", probeType), "must be integer")
			} else if portNum <= 0 || portNum >= 65536 {
				v.addError(port.Line, fmt.Sprintf("spec.containers[].%s.httpGet.port", probeType), "value out of range")
			}
		}
	}
}

func (v *Validator) validateResourceRequirements(node *yaml.Node) {
	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}
		key := node.Content[i]
		value := node.Content[i+1]
		fields[key.Value] = value
	}

	if requests, exists := fields["requests"]; exists {
		v.validateResourceObject(requests, "requests")
	}

	if limits, exists := fields["limits"]; exists {
		v.validateResourceObject(limits, "limits")
	}
}

func (v *Validator) validateResourceObject(node *yaml.Node, resourceType string) {
	memoryRegex := regexp.MustCompile(`^\d+(Gi|Mi|Ki)$`)

	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}
		key := node.Content[i]
		value := node.Content[i+1]
		fields[key.Value] = value
	}

	if cpu, exists := fields["cpu"]; exists {
		if cpu.Kind != yaml.ScalarNode {
			v.addError(cpu.Line, fmt.Sprintf("spec.containers[].resources.%s.cpu", resourceType), "must be integer")
		} else {
			if _, err := strconv.Atoi(cpu.Value); err != nil {
				v.addError(cpu.Line, fmt.Sprintf("spec.containers[].resources.%s.cpu", resourceType), "must be integer")
			}
		}
	}

	if memory, exists := fields["memory"]; exists {
		if memory.Kind != yaml.ScalarNode {
			v.addError(memory.Line, fmt.Sprintf("spec.containers[].resources.%s.memory", resourceType), "must be string")
		} else if !memoryRegex.MatchString(memory.Value) {
			v.addError(memory.Line, fmt.Sprintf("spec.containers[].resources.%s.memory", resourceType), fmt.Sprintf("has invalid format '%s'", memory.Value))
		}
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <yaml-file>\n", os.Args[0])
		os.Exit(1)
	}

	filePath := os.Args[1]
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	validator := &Validator{file: filePath}

	// Валидируем каждый документ в YAML файле
	for _, doc := range root.Content {
		if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
			validator.validateTopLevel(doc.Content[0])
		}
	}

	if len(validator.errors) > 0 {
		for _, err := range validator.errors {
			fmt.Fprintln(os.Stderr, err.String())
		}
		os.Exit(1)
	}

	fmt.Println("YAML validation successful!")
	os.Exit(0)
}