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
	err := run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Application failed with error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	newCommand := flag.NewFlagSet("new", flag.ExitOnError)
	newToken := newCommand.String("token", "", "stage the pipeline is currently in")
	newGroup := newCommand.String("group", "", "stage the pipeline is currently in")
	newApp := newCommand.String("app", "", "stage the pipeline is currently in")
	newTag := newCommand.String("tag", "", "stage the pipeline is currently in")

	promoteCommand := flag.NewFlagSet("promote", flag.ExitOnError)
	promoteToken := promoteCommand.String("token", "", "stage the pipeline is currently in")

	statusCommand := flag.NewFlagSet("status", flag.ExitOnError)
	statusToken := statusCommand.String("token", "", "stage the pipeline is currently in")

	if len(args) < 2 {
		return fmt.Errorf("new, promote or status subcommand is required")
	}

	path, err := setupFilesystem()
	if err != nil {
		return err
	}

	defer func() {
		err := os.Remove(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to remove path %q, returned error: %s", path, err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var commandErr error
	var message string
	switch args[1] {
	case "new":
		err := newCommand.Parse(args[2:])
		if err != nil {
			return err
		}
		message, commandErr = command.NewCommand(ctx, path, *newToken, *newGroup, *newApp, *newTag)
	case "promote":
		err := promoteCommand.Parse(args[2:])
		if err != nil {
			return err
		}
		message, commandErr = command.PromoteCommand(ctx, path, *promoteToken)
	case "status":
		err := statusCommand.Parse(args[2:])
		if err != nil {
			return err
		}
		message, commandErr = command.StatusCommand(ctx, path, *statusToken)
	default:
		flag.PrintDefaults()
		return fmt.Errorf("Unknown flag: %s", args[1])
	}

	if commandErr != nil {
		return commandErr
	}

	fmt.Println(message)

	return err
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
