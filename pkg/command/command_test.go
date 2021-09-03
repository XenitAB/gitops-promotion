package command

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

type providerConfig struct {
	providerType  string
	username      string
	password      string
	url           string
	defaultBranch string
}

var providers = []providerConfig{
	{
		providerType:  "azdo",
		username:      "gitops-promotion",
		password:      os.Getenv("AZDO_PAT"),
		url:           os.Getenv("AZDO_URL"),
		defaultBranch: "main",
	},
	{
		providerType:  "github",
		username:      "gitops-promotion",
		password:      os.Getenv("GITHUB_TOKEN"),
		url:           os.Getenv("GITHUB_URL"),
		defaultBranch: "main",
	},
}

func TestProviderE2E(t *testing.T) {
	for _, p := range providers {
		t.Run(p.providerType, func(t *testing.T) {
			if p.url == "" || p.password == "" {
				t.Skipf("Skipping test since url or password env var is not set")
			}
			path := t.TempDir()
			providerType, err := git.StringToProviderType(p.providerType)
			require.NoError(t, err)

			testCloneRepository(t, p.url, p.username, p.password, path, p.defaultBranch)

			now := time.Now()
			tag := now.Format("20060102150405")
			group := "testgroup"
			app := "testapp"

			promoteBranchName := fmt.Sprintf("promote/%s-%s", group, app)

			ctx := context.Background()

			// Test DEV
			newCommandMsgDev, err := NewCommand(ctx, p.providerType, path, p.password, group, app, tag)
			require.NoError(t, err)

			require.Equal(t, "created promotions pull request", newCommandMsgDev)

			path = testCloneRepositoryAndValidateTag(t, p.url, p.username, p.password, p.defaultBranch, group, "dev", app, tag)

			repoDev := testGetRepository(t, path)
			revDev := testGetRepositoryHeadRevision(t, repoDev)

			testSetStatus(t, ctx, providerType, revDev, group, "dev", p.url, p.password, true)

			// Test QA
			promoteCommandMsgQa, err := PromoteCommand(ctx, p.providerType, path, p.password)
			require.NoError(t, err)

			require.Equal(t, "created promotions pull request", promoteCommandMsgQa)

			path = testCloneRepositoryAndValidateTag(t, p.url, p.username, p.password, promoteBranchName, group, "qa", app, tag)
			statusCommandMsgQa, err := StatusCommand(ctx, p.providerType, path, p.password)
			require.NoError(t, err)

			require.Equal(t, "status check has succeed", statusCommandMsgQa)

			repoQa := testGetRepository(t, path)
			revQa := testGetRepositoryHeadRevision(t, repoQa)

			testMergePR(t, ctx, providerType, p.url, p.password, promoteBranchName, revQa)

			path = testCloneRepositoryAndValidateTag(t, p.url, p.username, p.password, p.defaultBranch, group, "qa", app, tag)

			repoMergedQa := testGetRepository(t, path)
			revMergedQa := testGetRepositoryHeadRevision(t, repoMergedQa)

			testSetStatus(t, ctx, providerType, revMergedQa, group, "qa", p.url, p.password, true)

			// Test PROD
			promoteCommandMsgProd, err := PromoteCommand(ctx, p.providerType, path, p.password)
			require.NoError(t, err)

			require.Equal(t, "created promotions pull request", promoteCommandMsgProd)

			path = testCloneRepositoryAndValidateTag(t, p.url, p.username, p.password, promoteBranchName, group, "prod", app, tag)
			statusCommandMsgProd, err := StatusCommand(ctx, p.providerType, path, p.password)
			require.NoError(t, err)

			require.Equal(t, "status check has succeed", statusCommandMsgProd)

			repoProd := testGetRepository(t, path)
			revProd := testGetRepositoryHeadRevision(t, repoProd)

			testMergePR(t, ctx, providerType, p.url, p.password, promoteBranchName, revProd)

			path = testCloneRepositoryAndValidateTag(t, p.url, p.username, p.password, p.defaultBranch, group, "prod", app, tag)

			repoMergedProd := testGetRepository(t, path)
			revMergedProd := testGetRepositoryHeadRevision(t, repoMergedProd)

			testSetStatus(t, ctx, providerType, revMergedProd, group, "prod", p.url, p.password, true)
		})
	}
}
