package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/xenitab/gitops-promotion/pkg/command"
)

func main() {
	newCommand := flag.NewFlagSet("new", flag.ExitOnError)
	newToken := newCommand.String("token", "", "stage the pipeline is currently in")
	newGroup := newCommand.String("group", "", "stage the pipeline is currently in")
	newApp := newCommand.String("app", "", "stage the pipeline is currently in")
	newTag := newCommand.String("tag", "", "stage the pipeline is currently in")

	promoteCommand := flag.NewFlagSet("promote", flag.ExitOnError)
	promoteToken := promoteCommand.String("token", "", "stage the pipeline is currently in")

	statusCommand := flag.NewFlagSet("status", flag.ExitOnError)
	statusToken := statusCommand.String("token", "", "stage the pipeline is currently in")

	if len(os.Args) < 2 {
		fmt.Println("new, promote, status or template subcommand is required")
		os.Exit(1)
		return
	}

	path, err := setupFilesystem()
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
		return
	}
	defer os.Remove(path)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var commandErr error
	var message string
	switch os.Args[1] {
	case "new":
		err := newCommand.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}
		message, commandErr = command.NewCommand(ctx, path, *newToken, *newGroup, *newApp, *newTag)
	case "promote":
		err := promoteCommand.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}
		message, commandErr = command.PromoteCommand(ctx, path, *promoteToken)
	case "status":
		err := statusCommand.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}
		message, commandErr = command.StatusCommand(ctx, path, *statusToken)
	default:
		flag.PrintDefaults()
		os.Exit(1)
		return
	}
	if commandErr != nil {
		fmt.Printf("ERROR: %v\n", commandErr)
		os.Exit(1)
		return
	}

	fmt.Println(message)
}

func setupFilesystem() (string, error) {
	curPath, err := os.Getwd()
	if err != nil {
		return "", err
	}

	tmpPath, err := os.MkdirTemp("", "gitops-promotion-")
	if err != nil {
		return "", err
	}

	err = fileutils.CopyDir(curPath, tmpPath, true, []string{})
	if err != nil {
		return "", err
	}

	return tmpPath, nil
}
