package command

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

func getConfig(path string) (config.Config, error) {
	file, err := os.Open(fmt.Sprintf("%s/gitops-promotion.yaml", path))
	if err != nil {
		return config.Config{}, err
	}
	cfg, err := config.LoadConfig(file)
	if err != nil {
		return config.Config{}, fmt.Errorf("could not load config: %w", err)
	}
	return cfg, nil
}

func getRepository(ctx context.Context, providerType string, path, token string) (*git.Repository, error) {
	repo, err := git.LoadRepository(ctx, path, providerType, token)
	if err != nil {
		return nil, fmt.Errorf("could not load repository: %w", err)
	}
	return repo, nil
}

func Run(args []string) (string, error) {
	defaultPath, err := os.Getwd()
	if err != nil {
		return "", err
	}

	newCommand := flag.NewFlagSet("new", flag.ExitOnError)
	newToken := newCommand.String("token", "", "Access token (PAT) to git provider")
	newGroup := newCommand.String("group", "", "Main application group")
	newApp := newCommand.String("app", "", "Name of the application")
	newTag := newCommand.String("tag", "", "Application version/tag to set")
	newProviderType := newCommand.String("provider", "azdo", "git provider to use")
	newPath := newCommand.String("sourcedir", defaultPath, "Source working tree to operate on")

	promoteCommand := flag.NewFlagSet("promote", flag.ExitOnError)
	promoteToken := promoteCommand.String("token", "", "Access token (PAT) to git provider")
	promoteProviderType := promoteCommand.String("provider", "azdo", "git provider to use")
	promotePath := promoteCommand.String("sourcedir", defaultPath, "Source working tree to operate on")

	statusCommand := flag.NewFlagSet("status", flag.ExitOnError)
	statusToken := statusCommand.String("token", "", "Access token (PAT) to git provider")
	statusProviderType := statusCommand.String("provider", "azdo", "git provider to use")
	statusPath := statusCommand.String("sourcedir", defaultPath, "Source working tree to operate on")

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
			return NewCommand(ctx, *newProviderType, workTreeCopy, *newToken, *newGroup, *newApp, *newTag)
		})
	case "promote":
		err := promoteCommand.Parse(args[2:])
		if err != nil {
			return "", err
		}
		message, commandErr = withCopyOfWorkTree(promotePath, func(workTreeCopy string) (string, error) {
			return PromoteCommand(ctx, *promoteProviderType, workTreeCopy, *promoteToken)
		})
	case "status":
		err := statusCommand.Parse(args[2:])
		if err != nil {
			return "", err
		}
		message, commandErr = withCopyOfWorkTree(statusPath, func(workTreeCopy string) (string, error) {
			return StatusCommand(ctx, *statusProviderType, workTreeCopy, *statusToken)
		})
	default:
		flag.PrintDefaults()
		return "", fmt.Errorf("Unknown flag: %s", args[1])
	}

	if commandErr != nil {
		return "", commandErr
	}

	return message, err
}

func withCopyOfWorkTree(sourcePath *string, work func(string) (string, error)) (string, error) {
	tmpPath, err := os.MkdirTemp("", "gitops-promotion-")
	if err != nil {
		return "", err
	}
	defer func() {
		err := fileutils.RemovePath(tmpPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to remove path %q, returned error: %s", tmpPath, err)
		}
	}()

	err = fileutils.CopyDir(*sourcePath, tmpPath, true, []string{})
	if err != nil {
		return "", err
	}

	message, err := work(tmpPath)
	return message, err
}
