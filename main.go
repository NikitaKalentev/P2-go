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
	Line    int
	Message string
}

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
	if err := validateFile(filename); err != nil {
		os.Exit(1)
	}
}

func validateFile(filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		return err
	}

	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		return err
	}

	errors := validateYAML(&root)
	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "%s:%d %s\n", filename, err.Line, err.Message)
		}
		return fmt.Errorf("validation failed")
	}

	return nil
}

func validateYAML(root *yaml.Node) []ValidationError {
	var errors []ValidationError

	if len(root.Content) == 0 {
		return append(errors, ValidationError{Line: 1, Message: "Empty YAML document"})
	}

	doc := root.Content[0]

	// Validate apiVersion
	apiVersionNode := findNode(doc, "apiVersion")
	if apiVersionNode == nil {
		errors = append(errors, ValidationError{Line: 1, Message: "apiVersion is required"})
	} else if apiVersionNode.Value != "v1" {
		errors = append(errors, ValidationError{
			Line:    apiVersionNode.Line,
			Message: fmt.Sprintf("apiVersion has unsupported value '%s'", apiVersionNode.Value),
		})
	}

	// Validate kind
	kindNode := findNode(doc, "kind")
	if kindNode == nil {
		errors = append(errors, ValidationError{Line: 1, Message: "kind is required"})
	} else if kindNode.Value != "Pod" {
		errors = append(errors, ValidationError{
			Line:    kindNode.Line,
			Message: fmt.Sprintf("kind has unsupported value '%s'", kindNode.Value),
		})
	}

	// Validate metadata
	metadataNode := findNode(doc, "metadata")
	if metadataNode == nil {
		errors = append(errors, ValidationError{Line: 1, Message: "metadata is required"})
	} else {
		nameNode := findNode(metadataNode, "name")
		if nameNode == nil {
			errors = append(errors, ValidationError{Line: metadataNode.Line, Message: "name is required"})
		} else if strings.TrimSpace(nameNode.Value) == "" {
			errors = append(errors, ValidationError{
				Line:    nameNode.Line,
				Message: "name is required",
			})
		}
	}

	// Validate spec
	specNode := findNode(doc, "spec")
	if specNode == nil {
		errors = append(errors, ValidationError{Line: 1, Message: "spec is required"})
	} else {
		// Validate OS
		osNode := findNode(specNode, "os")
		if osNode != nil && osNode.Value != "" {
			if osNode.Value != "linux" && osNode.Value != "windows" {
				errors = append(errors, ValidationError{
					Line:    osNode.Line,
					Message: fmt.Sprintf("os has unsupported value '%s'", osNode.Value),
				})
			}
		}

		// Validate containers
		containersNode := findNode(specNode, "containers")
		if containersNode == nil {
			errors = append(errors, ValidationError{Line: specNode.Line, Message: "containers is required"})
		} else if len(containersNode.Content) == 0 {
			errors = append(errors, ValidationError{Line: containersNode.Line, Message: "containers is required"})
		} else {
			for _, containerNode := range containersNode.Content {
				errors = append(errors, validateContainer(containerNode)...)
			}
		}
	}

	return errors
}

func validateContainer(containerNode *yaml.Node) []ValidationError {
	var errors []ValidationError

	// Validate container name
	nameNode := findNode(containerNode, "name")
	if nameNode == nil {
		errors = append(errors, ValidationError{Line: containerNode.Line, Message: "name is required"})
	} else if strings.TrimSpace(nameNode.Value) == "" {
		errors = append(errors, ValidationError{
			Line:    nameNode.Line,
			Message: "name is required",
		})
	} else if !snakeCaseRegex.MatchString(nameNode.Value) {
		errors = append(errors, ValidationError{
			Line:    nameNode.Line,
			Message: fmt.Sprintf("name has invalid format '%s'", nameNode.Value),
		})
	}

	// Validate container image
	imageNode := findNode(containerNode, "image")
	if imageNode == nil {
		errors = append(errors, ValidationError{Line: containerNode.Line, Message: "image is required"})
	} else if strings.TrimSpace(imageNode.Value) == "" {
		errors = append(errors, ValidationError{
			Line:    imageNode.Line,
			Message: "image is required",
		})
	} else if !imageRegex.MatchString(imageNode.Value) {
		errors = append(errors, ValidationError{
			Line:    imageNode.Line,
			Message: fmt.Sprintf("image has invalid format '%s'", imageNode.Value),
		})
	}

	// Validate resources
	resourcesNode := findNode(containerNode, "resources")
	if resourcesNode == nil {
		errors = append(errors, ValidationError{Line: containerNode.Line, Message: "resources is required"})
	} else {
		errors = append(errors, validateResources(resourcesNode)...)
	}

	// Validate ports if present
	portsNode := findNode(containerNode, "ports")
	if portsNode != nil {
		for _, portNode := range portsNode.Content {
			errors = append(errors, validatePort(portNode)...)
		}
	}

	// Validate probes if present
	if readinessProbeNode := findNode(containerNode, "readinessProbe"); readinessProbeNode != nil {
		errors = append(errors, validateProbe(readinessProbeNode)...)
	}
	if livenessProbeNode := findNode(containerNode, "livenessProbe"); livenessProbeNode != nil {
		errors = append(errors, validateProbe(livenessProbeNode)...)
	}

	return errors
}

func validatePort(portNode *yaml.Node) []ValidationError {
	var errors []ValidationError

	containerPortNode := findNode(portNode, "containerPort")
	if containerPortNode == nil {
		errors = append(errors, ValidationError{Line: portNode.Line, Message: "containerPort is required"})
	} else {
		port, err := strconv.Atoi(containerPortNode.Value)
		if err != nil {
			errors = append(errors, ValidationError{
				Line:    containerPortNode.Line,
				Message: "containerPort must be int",
			})
		} else if port <= 0 || port >= 65536 {
			errors = append(errors, ValidationError{
				Line:    containerPortNode.Line,
				Message: "containerPort value out of range",
			})
		}
	}

	if protocolNode := findNode(portNode, "protocol"); protocolNode != nil && protocolNode.Value != "" {
		if protocolNode.Value != "TCP" && protocolNode.Value != "UDP" {
			errors = append(errors, ValidationError{
				Line:    protocolNode.Line,
				Message: fmt.Sprintf("protocol has unsupported value '%s'", protocolNode.Value),
			})
		}
	}

	return errors
}

func validateProbe(probeNode *yaml.Node) []ValidationError {
	var errors []ValidationError

	httpGetNode := findNode(probeNode, "httpGet")
	if httpGetNode == nil {
		errors = append(errors, ValidationError{Line: probeNode.Line, Message: "httpGet is required"})
	} else {
		pathNode := findNode(httpGetNode, "path")
		if pathNode == nil {
			errors = append(errors, ValidationError{Line: httpGetNode.Line, Message: "path is required"})
		} else if strings.TrimSpace(pathNode.Value) == "" {
			errors = append(errors, ValidationError{
				Line:    pathNode.Line,
				Message: "path is required",
			})
		} else if !strings.HasPrefix(pathNode.Value, "/") {
			errors = append(errors, ValidationError{
				Line:    pathNode.Line,
				Message: fmt.Sprintf("path has invalid format '%s'", pathNode.Value),
			})
		}

		portNode := findNode(httpGetNode, "port")
		if portNode == nil {
			errors = append(errors, ValidationError{Line: httpGetNode.Line, Message: "port is required"})
		} else {
			port, err := strconv.Atoi(portNode.Value)
			if err != nil {
				errors = append(errors, ValidationError{
					Line:    portNode.Line,
					Message: "port must be int",
				})
			} else if port <= 0 || port >= 65536 {
				errors = append(errors, ValidationError{
					Line:    portNode.Line,
					Message: "port value out of range",
				})
			}
		}
	}

	return errors
}

func validateResources(resourcesNode *yaml.Node) []ValidationError {
	var errors []ValidationError

	if requestsNode := findNode(resourcesNode, "requests"); requestsNode != nil {
		errors = append(errors, validateResourceMap(requestsNode)...)
	}
	if limitsNode := findNode(resourcesNode, "limits"); limitsNode != nil {
		errors = append(errors, validateResourceMap(limitsNode)...)
	}

	return errors
}

func validateResourceMap(resourceMapNode *yaml.Node) []ValidationError {
	var errors []ValidationError

	for i := 0; i < len(resourceMapNode.Content); i += 2 {
		if i+1 >= len(resourceMapNode.Content) {
			break
		}
		keyNode := resourceMapNode.Content[i]
		valueNode := resourceMapNode.Content[i+1]

		switch keyNode.Value {
		case "cpu":
			if valueNode.Kind != yaml.ScalarNode {
				errors = append(errors, ValidationError{
					Line:    valueNode.Line,
					Message: "cpu must be int",
				})
			} else if _, err := strconv.Atoi(valueNode.Value); err != nil {
				errors = append(errors, ValidationError{
					Line:    valueNode.Line,
					Message: "cpu must be int",
				})
			}
		case "memory":
			if valueNode.Kind != yaml.ScalarNode {
				errors = append(errors, ValidationError{
					Line:    valueNode.Line,
					Message: "memory must be string",
				})
			} else if !memoryRegex.MatchString(valueNode.Value) {
				errors = append(errors, ValidationError{
					Line:    valueNode.Line,
					Message: fmt.Sprintf("memory has invalid format '%s'", valueNode.Value),
				})
			} else {
				numStr := valueNode.Value[:len(valueNode.Value)-2]
				if num, err := strconv.Atoi(numStr); err != nil || num < 0 {
					errors = append(errors, ValidationError{
						Line:    valueNode.Line,
						Message: "memory value out of range",
					})
				}
			}
		default:
			errors = append(errors, ValidationError{
				Line:    keyNode.Line,
				Message: fmt.Sprintf("%s has unsupported value", keyNode.Value),
			})
		}
	}

	return errors
}

func findNode(parent *yaml.Node, key string) *yaml.Node {
	if parent == nil || parent.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(parent.Content); i += 2 {
		if i+1 < len(parent.Content) {
			keyNode := parent.Content[i]
			valueNode := parent.Content[i+1]
			if keyNode.Value == key {
				return valueNode
			}
		}
	}
	return nil
}