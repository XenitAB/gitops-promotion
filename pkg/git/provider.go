package git

import (
	"context"
	"fmt"
)

type ProviderType string

const (
	ProviderTypeAzdo ProviderType = "azdo"
)

type GitProvider interface {
	GetStatus(ctx context.Context, sha, group, env string) (Status, error)
	CreatePR(ctx context.Context, branchName string, auto bool, state PRState) error
	GetPRWithBranch(ctx context.Context, source, target string) (PullRequest, error)
	GetPRThatCausedCommit(ctx context.Context, sha string) (PullRequest, error)
	MergePR(ctx context.Context, ID int, sha string) error
}

func NewGitProvider(ctx context.Context, providerType ProviderType, remoteURL, token string) (GitProvider, error) {
	switch providerType {
	case ProviderTypeAzdo:
		return NewAzdoGITProvider(ctx, remoteURL, token)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}
