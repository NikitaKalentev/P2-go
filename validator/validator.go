package validator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var path string

func Run(filename string) []error {
	absPath, _ := filepath.Abs(filename)
	parentDir := filepath.Dir(filename)
	relPath, _ := filepath.Rel(parentDir, filename)

	path = relPath

	content, err := os.ReadFile(absPath)

	if err != nil {
		pushErr(fmt.Errorf("cannot read file content: %w", err))
		return errs
	}

	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		pushErr(fmt.Errorf("cannot unmarshal file content: %w", err))
		return errs
	}

	for _, doc := range root.Content {
		validatePod(doc)
	}

	return errs
}

func validatePod(doc *yaml.Node) {
	required := []string{"apiVersion", "kind", "metadata", "spec"}
	validate(doc, required, func(key, val *yaml.Node) {
		switch key.Value {
		case "apiVersion":
			validateAPIVersion(val)
		case "kind":
			validateKind(val)
		case "metadata":
			validateMetadata(val)
		case "spec":
			validateSpec(val)
		}
	})
}

func validateMetadata(node *yaml.Node) {
	required := []string{"name"}
	validate(node, required, func(key, val *yaml.Node) {
		switch key.Value {
		case "name":
			validateName(val, false)
		case "labels":
			validateLabels(val)
		}
	})
}

func validateSpec(node *yaml.Node) {
	required := []string{"containers"}
	validate(node, required, func(key, val *yaml.Node) {
		switch key.Value {
		case "os":
			validateOS(val)
		case "containers":
			validateContainers(val)
		}
	})
}

func validateContainers(node *yaml.Node) {
	for _, container := range node.Content {
		validateContainer(container)
	}
}

func validateContainer(node *yaml.Node) {
	required := []string{"name", "image", "resources"}
	validate(node, required, func(key, val *yaml.Node) {
		switch key.Value {
		case "name":
			validateName(val, true)
		case "image":
			pattern := regexp.MustCompile(`^registry.bigbrother.io/(.*):(.*)$`)
			if !pattern.MatchString(val.Value) {
				pushErr(NewInvalidFormatError(key.Value, val.Value, key.Line))
			}
		case "ports":
			for _, port := range val.Content {
				validateContainerPort(port)
			}
		case "readinessProbe":
			validateProbe(val)
		case "livenessProbe":
			validateProbe(val)
		case "resources":
			validateResources(val)
		}
	})
}

func validateContainerPort(node *yaml.Node) {
	required := []string{"containerPort"}
	validate(node, required, func(key, val *yaml.Node) {
		switch key.Value {
		case "containerPort":
			validateOutOfRange(key, val, 0, 65535)
		case "protocol":
			if val.Value != "TCP" && val.Value != "UDP" {
				pushErr(NewUnsupportedValueError(key.Value, val.Value, key.Line))
			}
		}
	})
}

func validateProbe(node *yaml.Node) {
	required := []string{"httpGet"}
	validate(node, required, func(key, val *yaml.Node) {
		switch key.Value {
		case "httpGet":
			validateHTTPGet(val)
		}
	})
}

func validateHTTPGet(node *yaml.Node) {
	required := []string{"path", "port"}
	validate(node, required, func(key, val *yaml.Node) {
		switch key.Value {
		case "path":
			if !strings.HasPrefix(val.Value, "/") {
				pushErr(NewInvalidFormatError(key.Value, val.Value, key.Line))
			}
		case "port":
			validateOutOfRange(key, val, 0, 65535)
		}
	})
}

func validateResources(node *yaml.Node) {
	required := []string{}
	validate(node, required, func(key, val *yaml.Node) {
		switch key.Value {
		case "requests":
			validateResourceDeclaration(val)
		case "limits":
			validateResourceDeclaration(val)
		}
	})
}

func validateResourceDeclaration(node *yaml.Node) {
	required := []string{}
	validate(node, required, func(key, val *yaml.Node) {
		switch key.Value {
		case "cpu":
			validateOutOfRange(key, val, 1, math.MaxInt)
		case "memory":
			validateMemory(val)
		}
	})
}

func validateLabels(node *yaml.Node) {
	required := []string{}
	validate(node, required, func(key, val *yaml.Node) {
		if val.Kind != yaml.ScalarNode {
			pushErr(NewTypeMismatchError(key.Value, "string", key.Line))
		}
	})
}

func validateAPIVersion(node *yaml.Node) {
	if node.Value != "v1" {
		pushErr(NewUnsupportedValueError("apiVersion", node.Value, node.Line))
	}
}

func validateKind(node *yaml.Node) {
	if node.Value != "Pod" {
		pushErr(NewUnsupportedValueError("kind", node.Value, node.Line))
	}
}

func validateName(node *yaml.Node, checkSnakeCase bool) {
	if node.Value == "" {
		pushErr(NewRequiredFieldErrorWithLine("name", node.Line))
	} else {
		if checkSnakeCase {
			if !isValidSnakeCase(node.Value) {
				pushErr(NewInvalidFormatError("name", node.Value, node.Line))
			}
		}
	}
}

func validateOS(node *yaml.Node) {
	if node.Value != "linux" && node.Value != "windows" {
		pushErr(NewUnsupportedValueError("os", node.Value, node.Line))
	}
}

func validateOutOfRange(key *yaml.Node, val *yaml.Node, from, to int) {
	if val.Tag != "!!int" {
		pushErr(NewTypeMismatchError(key.Value, "int", key.Line))
	} else {
		number, _ := strconv.Atoi(val.Value)
		if number < from || number > to {
			pushErr(NewValueOutOfRangeError(key.Value, key.Line))
		}
	}
}

func validateMemory(node *yaml.Node) {
	pattern := regexp.MustCompile(`^(\d+)(Mi|Gi|Ki)$`)
	result := pattern.FindStringSubmatch(node.Value)
	if len(result) != 3 {
		pushErr(NewInvalidFormatError("memory", node.Value, node.Line))
	} else {
		amount, err := strconv.Atoi(result[1])
		if err != nil {
			pushErr(NewTypeMismatchError("memory", "int", node.Line))
		} else {
			if amount < 1 {
				pushErr(NewValueOutOfRangeError("memory", node.Line))
			}
		}
	}
}