package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/NikitaKalentev/P2-go/pod"
	"github.com/NikitaKalentev/P2-go/structure"
)

func main() {
	var code = 0
	var errors []string
	defer func() {
		os.Exit(code)
	}()

	if len(os.Args) <= 1 {
		errors = append(errors, "first argument is required")
	} else {
		filename := os.Args[1]
		if _, err := os.Stat(filename); err != nil {
			errors = append(errors, "could not read yaml file")
		} else {
			abspath, err := filepath.Abs(filename)
			if err == nil {
				filename = abspath
			}

			validator := structure.NewValidator(pod.NewPod())
			if validator.AcceptFile(filename) {
				validator.Validate()
			}
			errors = validator.GetErrors()
		}
	}

	if len(errors) > 0 {
		code = -1
		for _, text := range errors {
			fmt.Println(text)
		}
	}
}