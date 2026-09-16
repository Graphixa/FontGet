package main

import (
	"errors"
	"fmt"
	"fontget/cmd"
	"fontget/internal/shared"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		var displayed *shared.DisplayedError
		if !errors.As(err, &displayed) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
