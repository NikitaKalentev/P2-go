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
	File    string
	Line    int
	Message string
}

func (e ValidationError) String() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d %s", e.File, e.Line, e.Message)
	}
	return fmt.Sprintf("%s %s", e.File, e.Message)
}

var errors []ValidationError

// Константы для сообщений об ошибках
const (
	ErrAPIVersionRequired      = "apiVersion is required"
	ErrAPIVersionUnsupported   = "apiVersion has unsupported value '%s'"
	ErrKindRequired            = "kind is required"
	ErrKindUnsupported         = "kind has unsupported value '%s'"
	ErrMetadataRequired        = "metadata is required"
	ErrSpecRequired            = "spec is required"
	ErrNameRequired            = "name is required"
	ErrImageRequired           = "image is required"
	ErrResourcesRequired       = "resources is required"
	ErrContainersRequired      = "containers is required"
	ErrContainerPortRequired   = "containerPort is required"
	ErrHTTPGetRequired         = "httpGet is required"
	ErrPathRequired            = "path is required"
	ErrPortRequired            = "port is required"
	ErrCPUMustBeInt            = "cpu must be int"
	ErrMemoryInvalidFormat     = "memory has invalid format '%s'"
	ErrMemoryOutOfRange        = "memory value out of range"
	ErrContainerPortOutOfRange = "containerPort value out of range"
	ErrPortOutOfRange          = "port value out of range"
	ErrPortMustBePositive      = "port must be positive"
	ErrOSUnsupported           = "os has unsupported value '%s'"
	ErrNameInvalidFormat       = "name has invalid format '%s'"
	ErrImageInvalidFormat      = "image has invalid format '%s'"
	ErrPathInvalidFormat       = "path has invalid format '%s'"
	ErrProtocolUnsupported     = "protocol has unsupported value '%s'"
	ErrUnsupportedValue        = "%s has unsupported value"
	ErrInvalidStructure        = "Invalid YAML structure"
	ErrEmptyDocument           = "Empty YAML document"
	ErrMustBeMapping           = "must be a mapping"
	ErrMustBeList              = "must be a list"
	ErrResourceMapEmpty        = "resource map is empty"
)

// Регулярные выражения
var (
	// snakeCaseRegex: проверяет формат snake_case (строчные буквы и подчёркивания)
	snakeCaseRegex = regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)
	// imageRegex: проверяет формат образа (домен registry.bigbrother.io с тегом версии)
	imageRegex = regexp.MustCompile(`^registry\.bigbrother\.io/[^:]+:.+$`)
	// memoryRegex: проверяет формат памяти (число с суффиксом Gi, Mi, Ki)
	memoryRegex = regexp.MustCompile(`^[0-9]+(Gi|Mi|Ki)$`)
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: yamlvalid <path-to-yaml-file>\n")
		os.Exit(1)
	}

	filePath := os.Args[1]

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s cannot read file\n", filePath)
		os.Exit(1)
	}

	// Parse YAML
	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		fmt.Fprintf(os.Stderr, "%s invalid yaml format\n", filePath)
		os.Exit(1)
	}

	// Validate
	validatePod(&root, filePath)

	// Output errors
	if len(errors) > 0 {
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "%s\n", e)
		}
		os.Exit(1)
	}

	os.Exit(0)
}

func validatePod(root *yaml.Node, filePath string) {
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		errors = append(errors, ValidationError{File: filePath, Message: ErrEmptyDocument})
		return
	}

	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		errors = append(errors, ValidationError{File: filePath, Line: doc.Line, Message: ErrInvalidStructure})
		return
	}

	pod := parseMapping(doc)

	// Validate top-level fields
	if !hasField(pod, "apiVersion") {
		errors = append(errors, ValidationError{File: filePath, Message: ErrAPIVersionRequired})
	} else if apiVersion := getStringValue(pod, "apiVersion"); apiVersion != "v1" {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    getFieldLine(pod, "apiVersion"),
			Message: fmt.Sprintf(ErrAPIVersionUnsupported, apiVersion),
		})
	}

	if !hasField(pod, "kind") {
		errors = append(errors, ValidationError{File: filePath, Message: ErrKindRequired})
	} else if kind := getStringValue(pod, "kind"); kind != "Pod" {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    getFieldLine(pod, "kind"),
			Message: fmt.Sprintf(ErrKindUnsupported, kind),
		})
	}

	if !hasField(pod, "metadata") {
		errors = append(errors, ValidationError{File: filePath, Message: ErrMetadataRequired})
	} else {
		validateMetadata(pod["metadata"], filePath)
	}

	if !hasField(pod, "spec") {
		errors = append(errors, ValidationError{File: filePath, Message: ErrSpecRequired})
	} else {
		validateSpec(pod["spec"], filePath)
	}
}

func validateMetadata(node *yaml.Node, filePath string) {
	if node.Kind != yaml.MappingNode {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: fmt.Sprintf("metadata %s", ErrMustBeMapping),
		})
		return
	}

	metadata := parseMapping(node)

	if !hasField(metadata, "name") {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: ErrNameRequired,
		})
	} else {
		name := getStringValue(metadata, "name")
		if name == "" {
			errors = append(errors, ValidationError{
				File:    filePath,
				Line:    getFieldLine(metadata, "name"),
				Message: ErrNameRequired,
			})
		}
	}
}

func validateSpec(node *yaml.Node, filePath string) {
	if node.Kind != yaml.MappingNode {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: fmt.Sprintf("spec %s", ErrMustBeMapping),
		})
		return
	}

	spec := parseMapping(node)

	// Validate OS if present
	if hasField(spec, "os") {
		validateOS(spec["os"], filePath)
	}

	// Validate containers
	if !hasField(spec, "containers") {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: ErrContainersRequired,
		})
	} else {
		validateContainers(spec["containers"], filePath)
	}
}

func validateOS(node *yaml.Node, filePath string) {
	if node.Kind != yaml.MappingNode {
		return
	}

	osMap := parseMapping(node)

	if !hasField(osMap, "name") {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: "os.name is required",
		})
	} else {
		osName := getStringValue(osMap, "name")
		if osName != "linux" && osName != "windows" {
			errors = append(errors, ValidationError{
				File:    filePath,
				Line:    getFieldLine(osMap, "name"),
				Message: fmt.Sprintf(ErrOSUnsupported, osName),
			})
		}
	}
}

func validateContainers(node *yaml.Node, filePath string) {
	if node.Kind != yaml.SequenceNode {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: fmt.Sprintf("containers %s", ErrMustBeList),
		})
		return
	}

	if len(node.Content) == 0 {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: ErrContainersRequired,
		})
		return
	}

	containerNames := make(map[string]bool)

	for _, containerNode := range node.Content {
		if containerNode.Kind != yaml.MappingNode {
			errors = append(errors, ValidationError{
				File:    filePath,
				Line:    containerNode.Line,
				Message: fmt.Sprintf("container %s", ErrMustBeMapping),
			})
			continue
		}

		container := parseMapping(containerNode)

		// Validate name
		if !hasField(container, "name") {
			errors = append(errors, ValidationError{
				File:    filePath,
				Line:    containerNode.Line,
				Message: ErrNameRequired,
			})
		} else {
			name := getStringValue(container, "name")
			if name == "" {
				errors = append(errors, ValidationError{
					File:    filePath,
					Line:    getFieldLine(container, "name"),
					Message: ErrNameRequired,
				})
			} else if !snakeCaseRegex.MatchString(name) {
				errors = append(errors, ValidationError{
					File:    filePath,
					Line:    getFieldLine(container, "name"),
					Message: fmt.Sprintf(ErrNameInvalidFormat, name),
				})
			}
			if containerNames[name] {
				errors = append(errors, ValidationError{
					File:    filePath,
					Line:    getFieldLine(container, "name"),
					Message: fmt.Sprintf("containers.name must be unique, '%s' is duplicated", name),
				})
			}
			containerNames[name] = true
		}

		// Validate image
		if !hasField(container, "image") {
			errors = append(errors, ValidationError{
				File:    filePath,
				Line:    containerNode.Line,
				Message: ErrImageRequired,
			})
		} else {
			image := getStringValue(container, "image")
			if image == "" {
				errors = append(errors, ValidationError{
					File:    filePath,
					Line:    getFieldLine(container, "image"),
					Message: ErrImageRequired,
				})
			} else if !imageRegex.MatchString(image) {
				errors = append(errors, ValidationError{
					File:    filePath,
					Line:    getFieldLine(container, "image"),
					Message: fmt.Sprintf(ErrImageInvalidFormat, image),
				})
			}
		}

		// Validate ports if present
		if hasField(container, "ports") {
			validatePorts(container["ports"], filePath)
		}

		// Validate probes if present
		if hasField(container, "readinessProbe") {
			validateProbe(container["readinessProbe"], "readinessProbe", filePath)
		}
		if hasField(container, "livenessProbe") {
			validateProbe(container["livenessProbe"], "livenessProbe", filePath)
		}

		// Validate resources
		if !hasField(container, "resources") {
			errors = append(errors, ValidationError{
				File:    filePath,
				Line:    containerNode.Line,
				Message: ErrResourcesRequired,
			})
		} else {
			validateResources(container["resources"], filePath)
		}
	}
}

func validatePorts(node *yaml.Node, filePath string) {
	if node.Kind != yaml.SequenceNode {
		return
	}

	for _, portNode := range node.Content {
		if portNode.Kind != yaml.MappingNode {
			continue
		}

		port := parseMapping(portNode)

		if !hasField(port, "containerPort") {
			errors = append(errors, ValidationError{
				File:    filePath,
				Line:    portNode.Line,
				Message: ErrContainerPortRequired,
			})
		} else {
			portNode := port["containerPort"]
			if portNode.Tag != "!!int" {
				errors = append(errors, ValidationError{
					File:    filePath,
					Line:    getFieldLine(port, "containerPort"),
					Message: "containerPort must be int",
				})
			} else {
				portValue := getIntValue(port, "containerPort")
				if portValue <= 0 {
					errors = append(errors, ValidationError{
						File:    filePath,
						Line:    getFieldLine(port, "containerPort"),
						Message: "containerPort must be positive",
					})
				} else if portValue >= 65536 {
					errors = append(errors, ValidationError{
						File:    filePath,
						Line:    getFieldLine(port, "containerPort"),
						Message: ErrContainerPortOutOfRange,
					})
				}
			}
		}

		if hasField(port, "protocol") {
			protocol := getStringValue(port, "protocol")
			if protocol != "TCP" && protocol != "UDP" {
				errors = append(errors, ValidationError{
					File:    filePath,
					Line:    getFieldLine(port, "protocol"),
					Message: fmt.Sprintf(ErrProtocolUnsupported, protocol),
				})
			}
		}
	}
}

func validateProbe(node *yaml.Node, probeName string, filePath string) {
	if node.Kind != yaml.MappingNode {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: fmt.Sprintf("probe %s", ErrMustBeMapping),
		})
		return
	}

	probe := parseMapping(node)

	if !hasField(probe, "httpGet") {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: ErrHTTPGetRequired,
		})
	} else {
		validateHTTPGetAction(probe["httpGet"], probeName, filePath)
	}
}

func validateHTTPGetAction(node *yaml.Node, probeName string, filePath string) {
	if node.Kind != yaml.MappingNode {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: fmt.Sprintf("httpGet %s", ErrMustBeMapping),
		})
		return
	}

	action := parseMapping(node)

	if !hasField(action, "path") {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: ErrPathRequired,
		})
	} else {
		path := getStringValue(action, "path")
		if path == "" {
			errors = append(errors, ValidationError{
				File:    filePath,
				Line:    getFieldLine(action, "path"),
				Message: ErrPathRequired,
			})
		} else if !strings.HasPrefix(path, "/") {
			errors = append(errors, ValidationError{
				File:    filePath,
				Line:    getFieldLine(action, "path"),
				Message: fmt.Sprintf(ErrPathInvalidFormat, path),
			})
		}
	}

	if !hasField(action, "port") {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: ErrPortRequired,
		})
	} else {
		portNode := action["port"]
		if portNode.Tag != "!!int" {
			errors = append(errors, ValidationError{
				File:    filePath,
				Line:    getFieldLine(action, "port"),
				Message: "port must be int",
			})
		} else {
			portValue := getIntValue(action, "port")
			if portValue <= 0 {
				errors = append(errors, ValidationError{
					File:    filePath,
					Line:    getFieldLine(action, "port"),
					Message: ErrPortMustBePositive,
				})
			} else if portValue >= 65536 {
				errors = append(errors, ValidationError{
					File:    filePath,
					Line:    getFieldLine(action, "port"),
					Message: ErrPortOutOfRange,
				})
			}
		}
	}
}

func validateResources(node *yaml.Node, filePath string) {
	if node.Kind != yaml.MappingNode {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: fmt.Sprintf("resources %s", ErrMustBeMapping),
		})
		return
	}

	resources := parseMapping(node)

	if len(resources) == 0 {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: ErrResourceMapEmpty,
		})
		return
	}

	if hasField(resources, "requests") {
		validateResourceSpec(resources["requests"], "requests", filePath)
	}

	if hasField(resources, "limits") {
		validateResourceSpec(resources["limits"], "limits", filePath)
	}
}

func validateResourceSpec(node *yaml.Node, specName string, filePath string) {
	if node.Kind != yaml.MappingNode {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: fmt.Sprintf("resource map %s", ErrMustBeMapping),
		})
		return
	}

	if len(node.Content) == 0 {
		errors = append(errors, ValidationError{
			File:    filePath,
			Line:    node.Line,
			Message: ErrResourceMapEmpty,
		})
		return
	}

	spec := parseMapping(node)

	if hasField(spec, "cpu") {
		cpuNode := spec["cpu"]
		if cpuNode.Tag != "!!int" {
			errors = append(errors, ValidationError{
				File:    filePath,
				Line:    getFieldLine(spec, "cpu"),
				Message: ErrCPUMustBeInt,
			})
		} else {
			cpuValue := getIntValue(spec, "cpu")
			if cpuValue <= 0 {
				errors = append(errors, ValidationError{
					File:    filePath,
					Line:    getFieldLine(spec, "cpu"),
					Message: "containers.resources." + specName + ".cpu value out of range",
				})
			}
		}
	}

	if hasField(spec, "memory") {
		memory := getStringValue(spec, "memory")
		if !memoryRegex.MatchString(memory) {
			errors = append(errors, ValidationError{
				File:    filePath,
				Line:    getFieldLine(spec, "memory"),
				Message: fmt.Sprintf(ErrMemoryInvalidFormat, memory),
			})
		} else {
			numStr := memory[:len(memory)-2]
			if num, err := strconv.Atoi(numStr); err != nil || num <= 0 {
				errors = append(errors, ValidationError{
					File:    filePath,
					Line:    getFieldLine(spec, "memory"),
					Message: ErrMemoryOutOfRange,
				})
			}
		}
	}

	// Проверка на недопустимые поля
	for key := range spec {
		if key != "cpu" && key != "memory" {
			errors = append(errors, ValidationError{
				File:    filePath,
				Line:    getFieldLine(spec, key),
				Message: fmt.Sprintf(ErrUnsupportedValue, key),
			})
		}
	}
}

// Helper functions

func parseMapping(node *yaml.Node) map[string]*yaml.Node {
	result := make(map[string]*yaml.Node)
	if node.Kind != yaml.MappingNode {
		return result
	}

	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) {
			key := node.Content[i]
			value := node.Content[i+1]
			if key != nil && value != nil {
				result[key.Value] = value
			}
		}
	}
	return result
}

func hasField(m map[string]*yaml.Node, key string) bool {
	_, ok := m[key]
	return ok
}

func getStringValue(m map[string]*yaml.Node, key string) string {
	if node, ok := m[key]; ok && node != nil {
		return node.Value
	}
	return ""
}

func getIntValue(m map[string]*yaml.Node, key string) int {
	if node, ok := m[key]; ok && node != nil {
		val, _ := strconv.Atoi(node.Value)
		return val
	}
	return 0
}

func getFieldLine(m map[string]*yaml.Node, key string) int {
	if node, ok := m[key]; ok && node != nil {
		return node.Line
	}
	return 0
}