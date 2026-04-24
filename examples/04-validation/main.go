// Example 04-validation demonstrates gonfig's validation rules and error inspection.
// This example intentionally uses invalid defaults to trigger validation errors.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/nniel-ape/gonfig"
)

type Config struct {
	// Port must be between 1 and 65535, but default is 0 → triggers "required" and "min".
	Port int `default:"0" validate:"required,min=1,max=65535" description:"listen port"`

	// Environment must be one of the allowed values, but default is "staging" → triggers "oneof".
	Environment string `default:"staging" validate:"oneof=development production" description:"app environment"`

	// Workers must be between 1 and 16, but default is 100 → triggers "max".
	Workers int `default:"100" validate:"min=1,max=16" description:"number of workers"`

	// Name is required, but default is empty → triggers "required".
	Name string `default:"" validate:"required" description:"application name"`
}

func main() {
	var cfg Config

	err := gonfig.Load(&cfg)

	// Check if the error is a validation error using errors.Is.
	if errors.Is(err, gonfig.ErrValidation) {
		fmt.Println("Validation failed (detected via errors.Is)")
	}

	// Extract the structured ValidationError using errors.As.
	var ve *gonfig.ValidationError
	if errors.As(err, &ve) {
		fmt.Printf("\nFound %d validation error(s):\n\n", len(ve.Errors))

		for i, fe := range ve.Errors {
			fmt.Printf("  %d. Field:   %s\n", i+1, fe.Field)
			fmt.Printf("     Rule:    %s\n", fe.Rule)
			fmt.Printf("     Message: %s\n\n", fe.Message)
		}

		os.Exit(1)
	}

	if err != nil {
		fmt.Println("unexpected error:", err)
		os.Exit(1)
	}

	fmt.Println("Config is valid!")
}
