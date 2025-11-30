package main

import (
	"errors"
	"fmt"
	"os"

	"https://github.com/NikitaKalentev/P2-go/validator"
)

func main() {
	if len(os.Args) < 2 {
		panic("usage: yamlvalid <filename>")
	}

	filename := os.Args[1]

	_, err := os.Stat(filename)

	if errors.Is(err, os.ErrNotExist) {
		panic(fmt.Sprintf("%s does not exist", filename))
	}

	if errs := validator.Run(filename); len(errs) != 0 {
		for _, err := range errs {
			fmt.Println(err)
		}
	}
}