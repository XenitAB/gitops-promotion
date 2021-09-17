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
	message, err := run(os.Args)
	if message != "" {
		fmt.Println(message)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Application failed with error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) (string, error) {
	defaultPath, err := os.Getwd()
	if err != nil {
		return "", err
	}

	newCommand := flag.NewFlagSet("new", flag.ExitOnError)
	newToken := newCommand.String("token", "", "stage the pipeline is currently in")
	newGroup := newCommand.String("group", "", "stage the pipeline is currently in")
	newApp := newCommand.String("app", "", "stage the pipeline is currently in")
	newTag := newCommand.String("tag", "", "stage the pipeline is currently in")
	newPath := newCommand.String("sourcedir", defaultPath, "Source working tree to operate on")
	newProviderType := newCommand.String("provider", "azdo", "git provider to use")

	promoteCommand := flag.NewFlagSet("promote", flag.ExitOnError)
	promoteToken := promoteCommand.String("token", "", "stage the pipeline is currently in")
	promotePath := promoteCommand.String("sourcedir", defaultPath, "Source working tree to operate on")
	promoteProviderType := promoteCommand.String("provider", "azdo", "git provider to use")

	statusCommand := flag.NewFlagSet("status", flag.ExitOnError)
	statusToken := statusCommand.String("token", "", "stage the pipeline is currently in")
	statusPath := statusCommand.String("sourcedir", defaultPath, "Source working tree to operate on")
	statusProviderType := statusCommand.String("provider", "azdo", "git provider to use")

	if len(args) < 2 {
		return "", fmt.Errorf("new, promote or status subcommand is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var commandErr error
	var message string
	switch args[1] {
	case "new":
		err := newCommand.Parse(args[2:])
		if err != nil {
			return "", err
		}
		message, commandErr = withCopyOfWorkTree(newPath, func(workTreeCopy string) (string, error) {
			return command.NewCommand(ctx, *newProviderType, workTreeCopy, *newToken, *newGroup, *newApp, *newTag)
		})
	case "promote":
		err := promoteCommand.Parse(args[2:])
		if err != nil {
			return "", err
		}
		message, commandErr = withCopyOfWorkTree(promotePath, func(workTreeCopy string) (string, error) {
			return command.PromoteCommand(ctx, *promoteProviderType, workTreeCopy, *promoteToken)
		})
	case "status":
		err := statusCommand.Parse(args[2:])
		if err != nil {
			return "", err
		}
		message, commandErr = withCopyOfWorkTree(statusPath, func(workTreeCopy string) (string, error) {
			return command.StatusCommand(ctx, *statusProviderType, workTreeCopy, *statusToken)
		})
	default:
		flag.PrintDefaults()
		return "", fmt.Errorf("unknown flag: %s", args[1])
	}

	if commandErr != nil {
		return "", commandErr
	}

	return message, err
}

func withCopyOfWorkTree(sourcePath *string, taskFn func(string) (string, error)) (string, error) {
	tmpPath, err := os.MkdirTemp("", "gitops-promotion-")
	if err != nil {
		return "", err
	}
	defer func() {
		err := os.Remove(tmpPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to remove path %q, returned error: %s", tmpPath, err)
		}
	}()

	err = fileutils.CopyDir(*sourcePath, tmpPath, true, []string{})
	if err != nil {
		return "", err
	}

	message, err := taskFn(tmpPath)
	return message, err
}
