package validator

import (
	"regexp"

	"gopkg.in/yaml.v3"
)

func isValidSnakeCase(s string) bool {
	match, _ := regexp.MatchString(`^[a-z]+(_[a-z]+)*$`, s)
	return match
}

func checkRequiredFields(visited map[string]bool, required []string) {
	for _, field := range required {
		if !visited[field] {
			pushErr(NewRequiredFieldError(field))
		}
	}
}

func validate(node *yaml.Node, required []string, callback func(key, val *yaml.Node)) {
	// отмечаем посещения
	visited := make(map[string]bool)

	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		val := node.Content[i+1]

		callback(key, val)

		visited[key.Value] = true
	}

	checkRequiredFields(visited, required)
}