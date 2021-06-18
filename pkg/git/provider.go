package git

import (
	"context"
	"fmt"
	"strings"
)

type ProviderType string

const (
	ProviderTypeAzdo   ProviderType = "azdo"
	ProviderTypeGitHub ProviderType = "github"
)

type GitProvider interface {
	GetStatus(ctx context.Context, sha, group, env string) (Status, error)
	CreatePR(ctx context.Context, branchName string, auto bool, state *PRState) error
	GetPRWithBranch(ctx context.Context, source, target string) (PullRequest, error)
	GetPRThatCausedCommit(ctx context.Context, sha string) (PullRequest, error)
	MergePR(ctx context.Context, ID int, sha string) error
}

func NewGitProvider(ctx context.Context, providerType ProviderType, remoteURL, token string) (GitProvider, error) {
	switch providerType {
	case ProviderTypeAzdo:
		return NewAzdoGITProvider(ctx, remoteURL, token)
	case ProviderTypeGitHub:
		return NewGitHubGITProvider(ctx, remoteURL, token)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

func StringToProviderType(p string) (ProviderType, error) {
	switch strings.ToLower(p) {
	case "azdo":
		return ProviderTypeAzdo, nil
	case "github":
		return ProviderTypeGitHub, nil
	default:
		return "", fmt.Errorf("Unknown provider selected: %s", p)
	}
}
