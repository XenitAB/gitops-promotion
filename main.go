package main

import (
	"context"
	"fmt"
	"os"

	"github.com/xenitab/gitops-promotion/pkg/command"
)

func main() {
	message, err := command.Run(context.Background(), os.Args)
	if message != "" {
		fmt.Println(message)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Application failed with error: %v\n", err)
		os.Exit(1)
	}
}
