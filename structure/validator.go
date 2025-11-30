package structure

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Validator struct {
	NodeReport
	DocumentRoot *Node
	Container    NodeReporter
	filename     string
}

func (v *Validator) AcceptDocument(root *yaml.Node) bool {
	if len(root.Content) == 0 {
		v.AddError("file is empty")

		return false
	} else if len(root.Content[0].Content) == 0 {
		v.AddError("not a yaml file")

		return false
	}

	v.Container.Accept(nil, root.Content[0])

	return true
}

func (v *Validator) AcceptString(contents string) bool {
	var root yaml.Node
	err := yaml.Unmarshal([]byte(contents), &root)
	if err != nil {
		v.AddError(err.Error())
		return false
	}
	return v.AcceptDocument(&root)
}

func (v *Validator) AcceptFile(filename string) bool {
	v.filename = filename
	contents, err := os.ReadFile(filename)
	if err != nil {
		v.AddError(err.Error())
		return false
	}

	return v.AcceptString(string(contents))
}

func (v *Validator) Validate() {
	v.Container.Validate()
}

func (v Validator) GetErrors() []string {
	// @todo добавить сюда и свои ошибки (нет файла, не получилось де-сериализовать)
	var errors []string
	errors = append(errors, v.Container.GetErrors()...)
	errors = append(errors, v.Errors...)
	if v.filename != "" {
		for i, value := range errors {
			errors[i] = fmt.Sprintf("%s:%s", filepath.Base(v.filename), value)
		}
	}

	return errors
}

func NewValidator(container NodeReporter) *Validator {
	return &Validator{
		Container: container,
	}
}