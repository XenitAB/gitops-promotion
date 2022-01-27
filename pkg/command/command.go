package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	flag "github.com/spf13/pflag"

	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

const (
	configFileName = "gitops-promotion.yaml"
)

func Run(ctx context.Context, args []string) (string, error) {
	// Global flags
	if len(args) < 2 {
		return "", fmt.Errorf("new, feature, promote, or status subcommand is required")
	}
	defaultPath, err := os.Getwd()
	if err != nil {
		return "", err
	}
	global := flag.NewFlagSet(args[1], flag.ExitOnError)
	global.ParseErrorsWhitelist = flag.ParseErrorsWhitelist{UnknownFlags: true}
	token := global.String("token", "", "Access token (PAT) to git provider")
	providerType := global.String("provider", "azdo", "The git provider to use")
	path := global.String("sourcedir", defaultPath, "Source working tree to operate on")
	err = global.Parse(args[2:])
	if err != nil {
		return "", err
	}

	// Load configuration
	file, err := os.Open(filepath.Join(*path, configFileName))
	if err != nil {
		return "", err
	}
	cfg, err := config.LoadConfig(file)
	if err != nil {
		return "", fmt.Errorf("could not load config: %w", err)
	}

	// Load repository
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
	err = fileutils.CopyDir(*path, tmpPath, true, []string{})
	if err != nil {
		return "", err
	}
	repo, err := git.LoadRepository(ctx, tmpPath, *providerType, *token)
	if err != nil {
		return "", fmt.Errorf("could not load %s repository: %w", *providerType, err)
	}

	// Run Command
	switch args[1] {
	case "new":
		newCommand := flag.NewFlagSet(args[1], flag.ExitOnError)
		newCommand.ParseErrorsWhitelist = flag.ParseErrorsWhitelist{UnknownFlags: true}
		group := newCommand.String("group", "", "Main application group")
		app := newCommand.String("app", "", "Name of the application")
		tag := newCommand.String("tag", "", "Application version/tag to set")
		err := newCommand.Parse(args[2:])
		if err != nil {
			return "", err
		}
		return NewCommand(ctx, cfg, repo, *group, *app, *tag)
	case "feature":
		featureCommand := flag.NewFlagSet(args[1], flag.ContinueOnError)
		featureCommand.ParseErrorsWhitelist = flag.ParseErrorsWhitelist{UnknownFlags: true}
		group := featureCommand.String("group", "", "Main application group")
		app := featureCommand.String("app", "", "Name of the application")
		tag := featureCommand.String("tag", "", "Application version/tag to set")
		err := featureCommand.Parse(args[2:])
		if err != nil {
			return "", err
		}
		return FeatureCommand(ctx, cfg, repo, *group, *app, *tag)
	case "promote":
		return PromoteCommand(ctx, cfg, repo)
	case "status":
		return StatusCommand(ctx, cfg, repo)
	default:
		return "", fmt.Errorf("Unknown command: %s", args[1])
	}
}
