package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Pod struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       PodSpec  `yaml:"spec"`
}

type Metadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

type PodSpec struct {
	OS        string      `yaml:"os,omitempty"`
	Containers []Container `yaml:"containers"`
}

type Container struct {
	Name           string               `yaml:"name"`
	Image          string               `yaml:"image"`
	Ports          []ContainerPort      `yaml:"ports,omitempty"`
	ReadinessProbe *Probe               `yaml:"readinessProbe,omitempty"`
	LivenessProbe  *Probe               `yaml:"livenessProbe,omitempty"`
	Resources      ResourceRequirements `yaml:"resources"`
}

type ContainerPort struct {
	ContainerPort int    `yaml:"containerPort"`
	Protocol      string `yaml:"protocol,omitempty"`
}

type Probe struct {
	HTTPGet HTTPGetAction `yaml:"httpGet"`
}

type HTTPGetAction struct {
	Path string `yaml:"path"`
	Port int    `yaml:"port"`
}

type ResourceRequirements struct {
	Requests map[string]interface{} `yaml:"requests,omitempty"`
	Limits   map[string]interface{} `yaml:"limits,omitempty"`
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

	var pod Pod
	if err := yaml.Unmarshal(content, &pod); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		return err
	}

	return validatePod(pod)
}

func validatePod(pod Pod) error {
	// Validate apiVersion
	if pod.APIVersion == "" {
		fmt.Fprintf(os.Stderr, "apiVersion is required\n")
		return fmt.Errorf("validation failed")
	}
	if pod.APIVersion != "v1" {
		fmt.Fprintf(os.Stderr, "apiVersion has unsupported value '%s'\n", pod.APIVersion)
		return fmt.Errorf("validation failed")
	}

	// Validate kind
	if pod.Kind == "" {
		fmt.Fprintf(os.Stderr, "kind is required\n")
		return fmt.Errorf("validation failed")
	}
	if pod.Kind != "Pod" {
		fmt.Fprintf(os.Stderr, "kind has unsupported value '%s'\n", pod.Kind)
		return fmt.Errorf("validation failed")
	}

	// Validate metadata
	if pod.Metadata.Name == "" {
		fmt.Fprintf(os.Stderr, "metadata.name is required\n")
		return fmt.Errorf("validation failed")
	}

	// Validate spec
	if len(pod.Spec.Containers) == 0 {
		fmt.Fprintf(os.Stderr, "spec.containers is required\n")
		return fmt.Errorf("validation failed")
	}

	// Validate OS
	if pod.Spec.OS != "" && pod.Spec.OS != "linux" && pod.Spec.OS != "windows" {
		fmt.Fprintf(os.Stderr, "spec.os has unsupported value '%s'\n", pod.Spec.OS)
		return fmt.Errorf("validation failed")
	}

	// Validate containers
	for i, container := range pod.Spec.Containers {
		if err := validateContainer(container, i); err != nil {
			return err
		}
	}

	return nil
}

func validateContainer(container Container, index int) error {
	// Validate container name
	if container.Name == "" {
		fmt.Fprintf(os.Stderr, "spec.containers[%d].name is required\n", index)
		return fmt.Errorf("validation failed")
	}
	if !snakeCaseRegex.MatchString(container.Name) {
		fmt.Fprintf(os.Stderr, "spec.containers[%d].name has invalid format '%s'\n", index, container.Name)
		return fmt.Errorf("validation failed")
	}

	// Validate container image
	if container.Image == "" {
		fmt.Fprintf(os.Stderr, "spec.containers[%d].image is required\n", index)
		return fmt.Errorf("validation failed")
	}
	if !imageRegex.MatchString(container.Image) {
		fmt.Fprintf(os.Stderr, "spec.containers[%d].image has invalid format '%s'\n", index, container.Image)
		return fmt.Errorf("validation failed")
	}

	// Validate ports
	for i, port := range container.Ports {
		if port.ContainerPort <= 0 || port.ContainerPort >= 65536 {
			fmt.Fprintf(os.Stderr, "spec.containers[%d].ports[%d].containerPort value out of range\n", index, i)
			return fmt.Errorf("validation failed")
		}
		if port.Protocol != "" && port.Protocol != "TCP" && port.Protocol != "UDP" {
			fmt.Fprintf(os.Stderr, "spec.containers[%d].ports[%d].protocol has unsupported value '%s'\n", index, i, port.Protocol)
			return fmt.Errorf("validation failed")
		}
	}

	// Validate probes
	if container.ReadinessProbe != nil {
		if err := validateProbe(container.ReadinessProbe, index, "readinessProbe"); err != nil {
			return err
		}
	}
	if container.LivenessProbe != nil {
		if err := validateProbe(container.LivenessProbe, index, "livenessProbe"); err != nil {
			return err
		}
	}

	// Validate resources
	if err := validateResources(container.Resources, index); err != nil {
		return err
	}

	return nil
}

func validateProbe(probe *Probe, containerIndex int, probeType string) error {
	if probe.HTTPGet.Path == "" {
		fmt.Fprintf(os.Stderr, "spec.containers[%d].%s.httpGet.path is required\n", containerIndex, probeType)
		return fmt.Errorf("validation failed")
	}
	if !strings.HasPrefix(probe.HTTPGet.Path, "/") {
		fmt.Fprintf(os.Stderr, "spec.containers[%d].%s.httpGet.path has invalid format '%s'\n", containerIndex, probeType, probe.HTTPGet.Path)
		return fmt.Errorf("validation failed")
	}
	if probe.HTTPGet.Port <= 0 || probe.HTTPGet.Port >= 65536 {
		fmt.Fprintf(os.Stderr, "spec.containers[%d].%s.httpGet.port value out of range\n", containerIndex, probeType)
		return fmt.Errorf("validation failed")
	}
	return nil
}

func validateResources(resources ResourceRequirements, containerIndex int) error {
	// Validate requests
	if resources.Requests != nil {
		if err := validateResourceMap(resources.Requests, containerIndex, "requests"); err != nil {
			return err
		}
	}

	// Validate limits
	if resources.Limits != nil {
		if err := validateResourceMap(resources.Limits, containerIndex, "limits"); err != nil {
			return err
		}
	}

	return nil
}

func validateResourceMap(resourceMap map[string]interface{}, containerIndex int, resourceType string) error {
	for key, value := range resourceMap {
		switch key {
		case "cpu":
			if cpu, ok := value.(int); !ok {
				fmt.Fprintf(os.Stderr, "spec.containers[%d].resources.%s.cpu must be int\n", containerIndex, resourceType)
				return fmt.Errorf("validation failed")
			} else if cpu < 0 {
				fmt.Fprintf(os.Stderr, "spec.containers[%d].resources.%s.cpu value out of range\n", containerIndex, resourceType)
				return fmt.Errorf("validation failed")
			}
		case "memory":
			if memory, ok := value.(string); !ok {
				fmt.Fprintf(os.Stderr, "spec.containers[%d].resources.%s.memory must be string\n", containerIndex, resourceType)
				return fmt.Errorf("validation failed")
			} else if !memoryRegex.MatchString(memory) {
				fmt.Fprintf(os.Stderr, "spec.containers[%d].resources.%s.memory has invalid format '%s'\n", containerIndex, resourceType, memory)
				return fmt.Errorf("validation failed")
			} else {
				// Validate numeric part
				numStr := memory[:len(memory)-2]
				if num, err := strconv.Atoi(numStr); err != nil || num < 0 {
					fmt.Fprintf(os.Stderr, "spec.containers[%d].resources.%s.memory value out of range\n", containerIndex, resourceType)
					return fmt.Errorf("validation failed")
				}
			}
		default:
			fmt.Fprintf(os.Stderr, "spec.containers[%d].resources.%s.%s has unsupported value\n", containerIndex, resourceType, key)
			return fmt.Errorf("validation failed")
		}
	}
	return nil
}