package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

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

// Вспомогательная функция для создания ошибок с корректным номером строки
func newValidationError(node *yaml.Node, message string) ValidationError {
	line := node.Line
	// Если строка не определена, ищем в дочерних узлах
	if line == 0 && node.Content != nil && len(node.Content) > 0 {
		for _, child := range node.Content {
			if child != nil && child.Line > 0 {
				line = child.Line
				break
			}
		}
	}
	return ValidationError{Line: line, Message: message}
}

func validateYAML(root *yaml.Node) []ValidationError {
	var errors []ValidationError

	if len(root.Content) == 0 {
		return []ValidationError{{Line: 1, Message: ErrEmptyDocument}}
	}

	doc := root.Content[0]
	errors = append(errors, validateTopLevel(doc)...)

	return errors
}

func validateTopLevel(doc *yaml.Node) []ValidationError {
	var errors []ValidationError

	if doc.Kind != yaml.MappingNode {
		return []ValidationError{newValidationError(doc, ErrInvalidStructure)}
	}

	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(doc.Content); i += 2 {
		if i+1 < len(doc.Content) {
			key := doc.Content[i]
			value := doc.Content[i+1]
			if key != nil && value != nil {
				fields[key.Value] = value
			}
		}
	}

	// Явный порядок проверок для гарантии порядка ошибок
	if apiVersion, exists := fields["apiVersion"]; !exists || apiVersion == nil {
		errors = append(errors, ValidationError{Line: 1, Message: ErrAPIVersionRequired})
	} else if apiVersion.Value != "v1" {
		errors = append(errors, newValidationError(apiVersion, fmt.Sprintf(ErrAPIVersionUnsupported, apiVersion.Value)))
	}

	if kind, exists := fields["kind"]; !exists || kind == nil {
		errors = append(errors, ValidationError{Line: 1, Message: ErrKindRequired})
	} else if kind.Value != "Pod" {
		errors = append(errors, newValidationError(kind, fmt.Sprintf(ErrKindUnsupported, kind.Value)))
	}

	if metadata, exists := fields["metadata"]; !exists || metadata == nil {
		errors = append(errors, ValidationError{Line: 1, Message: ErrMetadataRequired})
	} else {
		errors = append(errors, validateMetadata(metadata)...)
	}

	if spec, exists := fields["spec"]; !exists || spec == nil {
		errors = append(errors, ValidationError{Line: 1, Message: ErrSpecRequired})
	} else {
		errors = append(errors, validateSpec(spec)...)
	}

	return errors
}

func validateMetadata(metadata *yaml.Node) []ValidationError {
	var errors []ValidationError

	if metadata.Kind != yaml.MappingNode {
		return []ValidationError{newValidationError(metadata, fmt.Sprintf("metadata %s", ErrMustBeMapping))}
	}

	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(metadata.Content); i += 2 {
		if i+1 < len(metadata.Content) {
			key := metadata.Content[i]
			value := metadata.Content[i+1]
			if key != nil && value != nil {
				fields[key.Value] = value
			}
		}
	}

	// Проверка имени в метаданных - исправлено: всегда используем узел name для ошибок
	if name, exists := fields["name"]; !exists || name == nil {
		errors = append(errors, newValidationError(metadata, ErrNameRequired))
	} else if strings.TrimSpace(name.Value) == "" {
		errors = append(errors, newValidationError(name, ErrNameRequired))
	}

	return errors
}

func validateSpec(spec *yaml.Node) []ValidationError {
	var errors []ValidationError

	if spec.Kind != yaml.MappingNode {
		return []ValidationError{newValidationError(spec, fmt.Sprintf("spec %s", ErrMustBeMapping))}
	}

	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(spec.Content); i += 2 {
		if i+1 < len(spec.Content) {
			key := spec.Content[i]
			value := spec.Content[i+1]
			if key != nil && value != nil {
				fields[key.Value] = value
			}
		}
	}

	// Явно заданный порядок проверок для гарантии порядка ошибок
	errors = append(errors, validateOS(fields)...)
	errors = append(errors, validateContainers(spec, fields)...)

	return errors
}

func validateOS(fields map[string]*yaml.Node) []ValidationError {
	var errors []ValidationError
	
	if os, exists := fields["os"]; exists && os != nil {
		if os.Value != "linux" && os.Value != "windows" {
			errors = append(errors, newValidationError(os, fmt.Sprintf(ErrOSUnsupported, os.Value)))
		}
	}
	
	return errors
}

func validateContainers(spec *yaml.Node, fields map[string]*yaml.Node) []ValidationError {
	var errors []ValidationError
	
	if containers, exists := fields["containers"]; !exists || containers == nil {
		errors = append(errors, newValidationError(spec, ErrContainersRequired))
	} else if containers.Kind != yaml.SequenceNode {
		errors = append(errors, newValidationError(containers, fmt.Sprintf("containers %s", ErrMustBeList)))
	} else if len(containers.Content) == 0 {
		errors = append(errors, newValidationError(containers, ErrContainersRequired))
	} else {
		for _, container := range containers.Content {
			if container != nil {
				errors = append(errors, validateContainer(container)...)
			}
		}
	}
	
	return errors
}

func validateContainer(container *yaml.Node) []ValidationError {
	var errors []ValidationError

	if container.Kind != yaml.MappingNode {
		return []ValidationError{newValidationError(container, fmt.Sprintf("container %s", ErrMustBeMapping))}
	}

	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(container.Content); i += 2 {
		if i+1 < len(container.Content) {
			key := container.Content[i]
			value := container.Content[i+1]
			if key != nil && value != nil {
				fields[key.Value] = value
			}
		}
	}

	// Явно заданный порядок проверок для гарантии порядка ошибок
	errors = append(errors, validateContainerName(fields)...)
	errors = append(errors, validateContainerImage(fields)...)
	errors = append(errors, validateContainerResources(fields)...)
	errors = append(errors, validateContainerPorts(fields)...)
	errors = append(errors, validateContainerProbes(fields)...)

	return errors
}

func validateContainerName(fields map[string]*yaml.Node) []ValidationError {
	var errors []ValidationError

	if name, exists := fields["name"]; !exists || name == nil {
		// Если поле name отсутствует, используем первый доступный узел для ошибки
		for _, node := range fields {
			errors = append(errors, newValidationError(node, ErrNameRequired))
			break
		}
		if len(fields) == 0 {
			errors = append(errors, ValidationError{Line: 1, Message: ErrNameRequired})
		}
	} else if strings.TrimSpace(name.Value) == "" {
		errors = append(errors, newValidationError(name, ErrNameRequired))
	} else if !snakeCaseRegex.MatchString(name.Value) {
		errors = append(errors, newValidationError(name, fmt.Sprintf(ErrNameInvalidFormat, name.Value)))
	}
	
	return errors
}

func validateContainerImage(fields map[string]*yaml.Node) []ValidationError {
	var errors []ValidationError

	if image, exists := fields["image"]; !exists || image == nil {
		// Если поле image отсутствует, используем первый доступный узел для ошибки
		for _, node := range fields {
			errors = append(errors, newValidationError(node, ErrImageRequired))
			break
		}
		if len(fields) == 0 {
			errors = append(errors, ValidationError{Line: 1, Message: ErrImageRequired})
		}
	} else if image.Value == "" {
		errors = append(errors, newValidationError(image, ErrImageRequired))
	} else if !imageRegex.MatchString(image.Value) {
		errors = append(errors, newValidationError(image, fmt.Sprintf(ErrImageInvalidFormat, image.Value)))
	}
	
	return errors
}

func validateContainerResources(fields map[string]*yaml.Node) []ValidationError {
	var errors []ValidationError

	if resources, exists := fields["resources"]; !exists || resources == nil {
		// Если поле resources отсутствует, используем первый доступный узел для ошибки
		for _, node := range fields {
			errors = append(errors, newValidationError(node, ErrResourcesRequired))
			break
		}
		if len(fields) == 0 {
			errors = append(errors, ValidationError{Line: 1, Message: ErrResourcesRequired})
		}
	} else {
		errors = append(errors, validateResourcesNode(resources)...)
	}
	
	return errors
}

func validateContainerPorts(fields map[string]*yaml.Node) []ValidationError {
	var errors []ValidationError

	if ports, exists := fields["ports"]; exists && ports != nil {
		if ports.Kind == yaml.SequenceNode {
			for _, port := range ports.Content {
				if port != nil {
					errors = append(errors, validatePort(port)...)
				}
			}
		}
	}
	
	return errors
}

func validateContainerProbes(fields map[string]*yaml.Node) []ValidationError {
	var errors []ValidationError

	// readinessProbe
	if probe, exists := fields["readinessProbe"]; exists && probe != nil {
		errors = append(errors, validateProbe(probe)...)
	}

	// livenessProbe
	if probe, exists := fields["livenessProbe"]; exists && probe != nil {
		errors = append(errors, validateProbe(probe)...)
	}
	
	return errors
}

func validateResourcesNode(resources *yaml.Node) []ValidationError {
	var errors []ValidationError

	if resources.Kind != yaml.MappingNode {
		return []ValidationError{newValidationError(resources, fmt.Sprintf("resources %s", ErrMustBeMapping))}
	}

	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(resources.Content); i += 2 {
		if i+1 < len(resources.Content) {
			key := resources.Content[i]
			value := resources.Content[i+1]
			if key != nil && value != nil {
				fields[key.Value] = value
			}
		}
	}

	if requests, exists := fields["requests"]; exists && requests != nil {
		errors = append(errors, validateResourceMap(requests)...)
	}

	if limits, exists := fields["limits"]; exists && limits != nil {
		errors = append(errors, validateResourceMap(limits)...)
	}

	return errors
}

func validateResourceMap(resourceMap *yaml.Node) []ValidationError {
	var errors []ValidationError

	if resourceMap.Kind != yaml.MappingNode {
		return []ValidationError{newValidationError(resourceMap, fmt.Sprintf("resource map %s", ErrMustBeMapping))}
	}

	// Проверка на пустой Content
	if len(resourceMap.Content) == 0 {
		return []ValidationError{newValidationError(resourceMap, ErrResourceMapEmpty)}
	}

	for i := 0; i < len(resourceMap.Content); i += 2 {
		if i+1 >= len(resourceMap.Content) {
			continue // защита от выхода за границы
		}
		
		key := resourceMap.Content[i]
		value := resourceMap.Content[i+1]

		if key == nil || value == nil {
			continue
		}

		switch key.Value {
		case "cpu":
			// КРИТИЧЕСКОЕ ИСПРАВЛЕНИЕ: Проверяем, что CPU - это число без кавычек
			if value.Value == "" {
				errors = append(errors, newValidationError(resourceMap, ErrCPUMustBeInt))
				continue
			}
			
			// Проверка тега YAML - должно быть целое число без кавычек
			if value.Tag != "!!int" {
				errors = append(errors, newValidationError(resourceMap, ErrCPUMustBeInt))
			} else {
				// Дополнительная проверка, что значение является числом
				if _, err := strconv.Atoi(value.Value); err != nil {
					errors = append(errors, newValidationError(resourceMap, ErrCPUMustBeInt))
				}
			}
			
		case "memory":
			if !memoryRegex.MatchString(value.Value) {
				errors = append(errors, newValidationError(value, fmt.Sprintf(ErrMemoryInvalidFormat, value.Value)))
			} else {
				numStr := value.Value[:len(value.Value)-2]
				if num, err := strconv.Atoi(numStr); err != nil || num <= 0 {
					errors = append(errors, newValidationError(value, ErrMemoryOutOfRange))
				}
			}
		default:
			errors = append(errors, newValidationError(key, fmt.Sprintf(ErrUnsupportedValue, key.Value)))
		}
	}

	return errors
}

func validatePort(port *yaml.Node) []ValidationError {
	var errors []ValidationError

	if port.Kind != yaml.MappingNode {
		return []ValidationError{newValidationError(port, fmt.Sprintf("port %s", ErrMustBeMapping))}
	}

	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(port.Content); i += 2 {
		if i+1 < len(port.Content) {
			key := port.Content[i]
			value := port.Content[i+1]
			if key != nil && value != nil {
				fields[key.Value] = value
			}
		}
	}

	if containerPort, exists := fields["containerPort"]; !exists || containerPort == nil {
		errors = append(errors, newValidationError(port, ErrContainerPortRequired))
	} else {
		portNum, err := strconv.Atoi(containerPort.Value)
		if err != nil {
			errors = append(errors, newValidationError(containerPort, "containerPort must be int"))
		} else {
			// КРИТИЧЕСКОЕ ИСПРАВЛЕНИЕ: Раздельная проверка отрицательных значений и значений вне диапазона
			if portNum <= 0 {
				errors = append(errors, newValidationError(containerPort, "containerPort must be positive"))
			} else if portNum >= 65536 {
				errors = append(errors, newValidationError(containerPort, ErrContainerPortOutOfRange))
			}
		}
	}

	if protocol, exists := fields["protocol"]; exists && protocol != nil && protocol.Value != "" {
		if protocol.Value != "TCP" && protocol.Value != "UDP" {
			errors = append(errors, newValidationError(protocol, fmt.Sprintf(ErrProtocolUnsupported, protocol.Value)))
		}
	}

	return errors
}

func validateProbe(probe *yaml.Node) []ValidationError {
	var errors []ValidationError

	if probe.Kind != yaml.MappingNode {
		return []ValidationError{newValidationError(probe, fmt.Sprintf("probe %s", ErrMustBeMapping))}
	}

	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(probe.Content); i += 2 {
		if i+1 < len(probe.Content) {
			key := probe.Content[i]
			value := probe.Content[i+1]
			if key != nil && value != nil {
				fields[key.Value] = value
			}
		}
	}

	if httpGet, exists := fields["httpGet"]; !exists || httpGet == nil {
		errors = append(errors, newValidationError(probe, ErrHTTPGetRequired))
	} else {
		errors = append(errors, validateHTTPGet(httpGet)...)
	}

	return errors
}

func validateHTTPGet(httpGet *yaml.Node) []ValidationError {
	var errors []ValidationError

	if httpGet.Kind != yaml.MappingNode {
		return []ValidationError{newValidationError(httpGet, fmt.Sprintf("httpGet %s", ErrMustBeMapping))}
	}

	fields := make(map[string]*yaml.Node)
	for i := 0; i < len(httpGet.Content); i += 2 {
		if i+1 < len(httpGet.Content) {
			key := httpGet.Content[i]
			value := httpGet.Content[i+1]
			if key != nil && value != nil {
				fields[key.Value] = value
			}
		}
	}

	if path, exists := fields["path"]; !exists || path == nil {
		errors = append(errors, newValidationError(httpGet, ErrPathRequired))
	} else if path.Value == "" {
		errors = append(errors, newValidationError(path, ErrPathRequired))
	} else if !strings.HasPrefix(path.Value, "/") {
		errors = append(errors, newValidationError(path, fmt.Sprintf(ErrPathInvalidFormat, path.Value)))
	}

	if port, exists := fields["port"]; !exists || port == nil {
		errors = append(errors, newValidationError(httpGet, ErrPortRequired))
	} else {
		portNum, err := strconv.Atoi(port.Value)
		if err != nil {
			errors = append(errors, newValidationError(port, "port must be int"))
		} else {
			// КРИТИЧЕСКОЕ ИСПРАВЛЕНИЕ: Явная раздельная проверка отрицательных значений и значений вне диапазона
			if portNum <= 0 {
				errors = append(errors, newValidationError(port, ErrPortMustBePositive))
			} else if portNum >= 65536 {
				errors = append(errors, newValidationError(port, ErrPortOutOfRange))
			}
		}
	}

	return errors
}