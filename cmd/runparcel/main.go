package main

import (
	"fmt"
	"os"

	"github.com/runparcel/runparcel/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
