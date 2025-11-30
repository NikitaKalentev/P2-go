package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <path-to-yaml-file>\n", os.Args[0])
		os.Exit(1)
	}

	filepath := os.Args[1]

	content, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read file: %v\n", filepath, err)
		os.Exit(1)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot parse YAML: %v\n", filepath, err)
		os.Exit(1)
	}

	if err := ValidatePod(filepath, &root); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	os.Exit(0)
}