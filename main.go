package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	snakeCaseRegex = regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)
	imageRegex     = regexp.MustCompile(`^registry\.bigbrother\.io/[^:]+:.+$`)
	memoryRegex    = regexp.MustCompile(`^[0-9]+(Gi|Mi|Ki)$`)
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <yaml-file>\n", os.Args[0])
		os.Exit(1)
	}

	filename := os.Args[1]
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	errors := validateYAML(&root)
	if len(errors) > 0 {
		// ВАЖНО: Выводим ошибки в stderr
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "%s:%d %s\n", filename, err.Line, err.Message)
		}
		os.Exit(1)
	}
}

type ValidationError struct {
	Line    int
	Message string
}

func validateYAML(root *yaml.Node) []ValidationError {
	var errors []ValidationError

	if len(root.Content) == 0 {
		return []ValidationError{{Line: 1, Message: "Empty YAML document"}}
	}

	doc := root.Content[0]
	errors = append(errors, validateTopLevel(doc)...)

	return errors
}

func validateTopLevel(doc *yaml.Node) []ValidationError {
	var errors []ValidationError

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(doc.Content); i += 2 {
		if i+1 < len(doc.Content) {
			key := doc.Content[i]
			value := doc.Content[i+1]
			fields[key.Value] = value
		}
	}

	// apiVersion
	if apiVersion, exists := fields["apiVersion"]; !exists {
		errors = append(errors, ValidationError{Line: 1, Message: "apiVersion is required"})
	} else if apiVersion.Value != "v1" {
		errors = append(errors, ValidationError{Line: apiVersion.Line, Message: fmt.Sprintf("apiVersion has unsupported value '%s'", apiVersion.Value)})
	}

	// kind
	if kind, exists := fields["kind"]; !exists {
		errors = append(errors, ValidationError{Line: 1, Message: "kind is required"})
	} else if kind.Value != "Pod" {
		errors = append(errors, ValidationError{Line: kind.Line, Message: fmt.Sprintf("kind has unsupported value '%s'", kind.Value)})
	}

	// metadata
	if metadata, exists := fields["metadata"]; !exists {
		errors = append(errors, ValidationError{Line: 1, Message: "metadata is required"})
	} else {
		errors = append(errors, validateMetadata(metadata)...)
	}

	// spec
	if spec, exists := fields["spec"]; !exists {
		errors = append(errors, ValidationError{Line: 1, Message: "spec is required"})
	} else {
		errors = append(errors, validateSpec(spec)...)
	}

	return errors
}

func validateMetadata(metadata *yaml.Node) []ValidationError {
	var errors []ValidationError

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(metadata.Content); i += 2 {
		if i+1 < len(metadata.Content) {
			key := metadata.Content[i]
			value := metadata.Content[i+1]
			fields[key.Value] = value
		}
	}

	if name, exists := fields["name"]; !exists {
		errors = append(errors, ValidationError{Line: metadata.Line, Message: "name is required"})
	} else if name.Value == "" {
		errors = append(errors, ValidationError{Line: name.Line, Message: "name is required"})
	}

	return errors
}

func validateSpec(spec *yaml.Node) []ValidationError {
	var errors []ValidationError

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(spec.Content); i += 2 {
		if i+1 < len(spec.Content) {
			key := spec.Content[i]
			value := spec.Content[i+1]
			fields[key.Value] = value
		}
	}

	// os
	if os, exists := fields["os"]; exists && os.Value != "" {
		if os.Value != "linux" && os.Value != "windows" {
			errors = append(errors, ValidationError{Line: os.Line, Message: fmt.Sprintf("os has unsupported value '%s'", os.Value)})
		}
	}

	// containers
	if containers, exists := fields["containers"]; !exists {
		errors = append(errors, ValidationError{Line: spec.Line, Message: "containers is required"})
	} else if len(containers.Content) == 0 {
		errors = append(errors, ValidationError{Line: containers.Line, Message: "containers is required"})
	} else {
		for _, container := range containers.Content {
			errors = append(errors, validateContainer(container)...)
		}
	}

	return errors
}

func validateContainer(container *yaml.Node) []ValidationError {
	var errors []ValidationError

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(container.Content); i += 2 {
		if i+1 < len(container.Content) {
			key := container.Content[i]
			value := container.Content[i+1]
			fields[key.Value] = value
		}
	}

	// name
	if name, exists := fields["name"]; !exists {
		errors = append(errors, ValidationError{Line: container.Line, Message: "name is required"})
	} else if name.Value == "" {
		errors = append(errors, ValidationError{Line: name.Line, Message: "name is required"})
	} else if !snakeCaseRegex.MatchString(name.Value) {
		errors = append(errors, ValidationError{Line: name.Line, Message: fmt.Sprintf("name has invalid format '%s'", name.Value)})
	}

	// image
	if image, exists := fields["image"]; !exists {
		errors = append(errors, ValidationError{Line: container.Line, Message: "image is required"})
	} else if image.Value == "" {
		errors = append(errors, ValidationError{Line: image.Line, Message: "image is required"})
	} else if !imageRegex.MatchString(image.Value) {
		errors = append(errors, ValidationError{Line: image.Line, Message: fmt.Sprintf("image has invalid format '%s'", image.Value)})
	}

	// resources
	if resources, exists := fields["resources"]; !exists {
		errors = append(errors, ValidationError{Line: container.Line, Message: "resources is required"})
	} else {
		errors = append(errors, validateResources(resources)...)
	}

	// ports
	if ports, exists := fields["ports"]; exists {
		for _, port := range ports.Content {
			errors = append(errors, validatePort(port)...)
		}
	}

	// readinessProbe
	if probe, exists := fields["readinessProbe"]; exists {
		errors = append(errors, validateProbe(probe)...)
	}

	// livenessProbe
	if probe, exists := fields["livenessProbe"]; exists {
		errors = append(errors, validateProbe(probe)...)
	}

	return errors
}

func validateResources(resources *yaml.Node) []ValidationError {
	var errors []ValidationError

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(resources.Content); i += 2 {
		if i+1 < len(resources.Content) {
			key := resources.Content[i]
			value := resources.Content[i+1]
			fields[key.Value] = value
		}
	}

	if requests, exists := fields["requests"]; exists {
		errors = append(errors, validateResourceMap(requests, "requests")...)
	}

	if limits, exists := fields["limits"]; exists {
		errors = append(errors, validateResourceMap(limits, "limits")...)
	}

	return errors
}

func validateResourceMap(resourceMap *yaml.Node, resourceType string) []ValidationError {
	var errors []ValidationError

	for i := 0; i < len(resourceMap.Content); i += 2 {
		if i+1 < len(resourceMap.Content) {
			key := resourceMap.Content[i]
			value := resourceMap.Content[i+1]

			switch key.Value {
			case "cpu":
				if _, err := strconv.Atoi(value.Value); err != nil {
					errors = append(errors, ValidationError{Line: value.Line, Message: "cpu must be int"})
				}
			case "memory":
				if !memoryRegex.MatchString(value.Value) {
					errors = append(errors, ValidationError{Line: value.Line, Message: fmt.Sprintf("memory has invalid format '%s'", value.Value)})
				} else {
					numStr := value.Value[:len(value.Value)-2]
					if num, err := strconv.Atoi(numStr); err != nil || num < 0 {
						errors = append(errors, ValidationError{Line: value.Line, Message: "memory value out of range"})
					}
				}
			default:
				errors = append(errors, ValidationError{Line: key.Line, Message: fmt.Sprintf("%s has unsupported value", key.Value)})
			}
		}
	}

	return errors
}

func validatePort(port *yaml.Node) []ValidationError {
	var errors []ValidationError

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(port.Content); i += 2 {
		if i+1 < len(port.Content) {
			key := port.Content[i]
			value := port.Content[i+1]
			fields[key.Value] = value
		}
	}

	if containerPort, exists := fields["containerPort"]; !exists {
		errors = append(errors, ValidationError{Line: port.Line, Message: "containerPort is required"})
	} else {
		portNum, err := strconv.Atoi(containerPort.Value)
		if err != nil {
			errors = append(errors, ValidationError{Line: containerPort.Line, Message: "containerPort must be int"})
		} else if portNum <= 0 || portNum >= 65536 {
			errors = append(errors, ValidationError{Line: containerPort.Line, Message: "containerPort value out of range"})
		}
	}

	if protocol, exists := fields["protocol"]; exists && protocol.Value != "" {
		if protocol.Value != "TCP" && protocol.Value != "UDP" {
			errors = append(errors, ValidationError{Line: protocol.Line, Message: fmt.Sprintf("protocol has unsupported value '%s'", protocol.Value)})
		}
	}

	return errors
}

func validateProbe(probe *yaml.Node) []ValidationError {
	var errors []ValidationError

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(probe.Content); i += 2 {
		if i+1 < len(probe.Content) {
			key := probe.Content[i]
			value := probe.Content[i+1]
			fields[key.Value] = value
		}
	}

	if httpGet, exists := fields["httpGet"]; !exists {
		errors = append(errors, ValidationError{Line: probe.Line, Message: "httpGet is required"})
	} else {
		errors = append(errors, validateHTTPGet(httpGet)...)
	}

	return errors
}

func validateHTTPGet(httpGet *yaml.Node) []ValidationError {
	var errors []ValidationError

	fields := map[string]*yaml.Node{}
	for i := 0; i < len(httpGet.Content); i += 2 {
		if i+1 < len(httpGet.Content) {
			key := httpGet.Content[i]
			value := httpGet.Content[i+1]
			fields[key.Value] = value
		}
	}

	if path, exists := fields["path"]; !exists {
		errors = append(errors, ValidationError{Line: httpGet.Line, Message: "path is required"})
	} else if path.Value == "" {
		errors = append(errors, ValidationError{Line: path.Line, Message: "path is required"})
	} else if !strings.HasPrefix(path.Value, "/") {
		errors = append(errors, ValidationError{Line: path.Line, Message: fmt.Sprintf("path has invalid format '%s'", path.Value)})
	}

	if port, exists := fields["port"]; !exists {
		errors = append(errors, ValidationError{Line: httpGet.Line, Message: "port is required"})
	} else {
		portNum, err := strconv.Atoi(port.Value)
		if err != nil {
			errors = append(errors, ValidationError{Line: port.Line, Message: "port must be int"})
		} else if portNum <= 0 || portNum >= 65536 {
			errors = append(errors, ValidationError{Line: port.Line, Message: "port value out of range"})
		}
	}

	return errors
}