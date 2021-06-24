package command

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

func TestE2EAzureDevOps(t *testing.T) {
	username := "gitops-promotion"
	password := testGetEnvOrSkip(t, "AZDO_PAT")
	url := testGetEnvOrSkip(t, "AZDO_URL")
	defaultBranch := "main"
	path := t.TempDir()

	providerTypeString := "azdo"
	providerType, err := git.StringToProviderType(providerTypeString)
	require.NoError(t, err)

	testCloneRepository(t, url, username, password, path, defaultBranch)

	now := time.Now()
	tag := now.Format("20060102150405")
	group := "testgroup"
	app := "testapp"

	promoteBranchName := fmt.Sprintf("promote/%s-%s", group, app)

	ctx := context.Background()

	// Test DEV
	newCommandMsgDev, err := NewCommand(ctx, providerTypeString, path, password, group, app, tag)
	require.NoError(t, err)

	require.Equal(t, "created promotions pull request", newCommandMsgDev)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, defaultBranch, group, "dev", app, tag)

	repoDev := testGetRepository(t, path)
	revDev := testGetRepositoryHeadRevision(t, repoDev)

	testSetStatus(t, ctx, providerType, revDev, group, "dev", url, password, true)

	// Test QA
	promoteCommandMsgQa, err := PromoteCommand(ctx, providerTypeString, path, password)
	require.NoError(t, err)

	require.Equal(t, "created promotions pull request", promoteCommandMsgQa)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, promoteBranchName, group, "qa", app, tag)
	statusCommandMsgQa, err := StatusCommand(ctx, providerTypeString, path, password)
	require.NoError(t, err)

	require.Equal(t, "status check has succeed", statusCommandMsgQa)

	repoQa := testGetRepository(t, path)
	revQa := testGetRepositoryHeadRevision(t, repoQa)

	testMergePR(t, ctx, providerType, url, password, promoteBranchName, revQa)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, defaultBranch, group, "qa", app, tag)

	repoMergedQa := testGetRepository(t, path)
	revMergedQa := testGetRepositoryHeadRevision(t, repoMergedQa)

	testSetStatus(t, ctx, providerType, revMergedQa, group, "qa", url, password, true)

	// Test PROD
	promoteCommandMsgProd, err := PromoteCommand(ctx, providerTypeString, path, password)
	require.NoError(t, err)

	require.Equal(t, "created promotions pull request", promoteCommandMsgProd)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, promoteBranchName, group, "prod", app, tag)
	statusCommandMsgProd, err := StatusCommand(ctx, providerTypeString, path, password)
	require.NoError(t, err)

	require.Equal(t, "status check has succeed", statusCommandMsgProd)

	repoProd := testGetRepository(t, path)
	revProd := testGetRepositoryHeadRevision(t, repoProd)

	testMergePR(t, ctx, providerType, url, password, promoteBranchName, revProd)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, defaultBranch, group, "prod", app, tag)

	repoMergedProd := testGetRepository(t, path)
	revMergedProd := testGetRepositoryHeadRevision(t, repoMergedProd)

	testSetStatus(t, ctx, providerType, revMergedProd, group, "prod", url, password, true)
}

func TestE2EGitHub(t *testing.T) {
	username := "gitops-promotion"
	password := testGetEnvOrSkip(t, "GITHUB_TOKEN")
	url := testGetEnvOrSkip(t, "GITHUB_URL")
	defaultBranch := "main"
	path := t.TempDir()

	providerTypeString := "github"
	providerType, err := git.StringToProviderType(providerTypeString)
	require.NoError(t, err)

	testCloneRepository(t, url, username, password, path, defaultBranch)

	now := time.Now()
	tag := now.Format("20060102150405")
	group := "testgroup"
	app := "testapp"

	promoteBranchName := fmt.Sprintf("promote/%s-%s", group, app)

	ctx := context.Background()

	// Test DEV
	newCommandMsgDev, err := NewCommand(ctx, providerTypeString, path, password, group, app, tag)
	require.NoError(t, err)

	require.Equal(t, "created promotions pull request", newCommandMsgDev)

	// TODO: Remove when auto merge is enabled in GitHub
	// START - Fake auto merge in GitHub
	pathFake := testCloneRepositoryAndValidateTag(t, url, username, password, promoteBranchName, group, "dev", app, tag)
	repoDevFake := testGetRepository(t, pathFake)
	revDevFake := testGetRepositoryHeadRevision(t, repoDevFake)

	testMergePR(t, ctx, providerType, url, password, promoteBranchName, revDevFake)
	// STOP - Fake auto merge in GitHub

	path = testCloneRepositoryAndValidateTag(t, url, username, password, defaultBranch, group, "dev", app, tag)

	repoDev := testGetRepository(t, path)
	revDev := testGetRepositoryHeadRevision(t, repoDev)

	testSetStatus(t, ctx, providerType, revDev, group, "dev", url, password, true)

	// Test QA
	promoteCommandMsgQa, err := PromoteCommand(ctx, providerTypeString, path, password)
	require.NoError(t, err)

	require.Equal(t, "created promotions pull request", promoteCommandMsgQa)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, promoteBranchName, group, "qa", app, tag)
	statusCommandMsgQa, err := StatusCommand(ctx, providerTypeString, path, password)
	require.NoError(t, err)

	require.Equal(t, "status check has succeed", statusCommandMsgQa)

	repoQa := testGetRepository(t, path)
	revQa := testGetRepositoryHeadRevision(t, repoQa)

	testMergePR(t, ctx, providerType, url, password, promoteBranchName, revQa)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, defaultBranch, group, "qa", app, tag)

	repoMergedQa := testGetRepository(t, path)
	revMergedQa := testGetRepositoryHeadRevision(t, repoMergedQa)

	testSetStatus(t, ctx, providerType, revMergedQa, group, "qa", url, password, true)

	// Test PROD
	promoteCommandMsgProd, err := PromoteCommand(ctx, providerTypeString, path, password)
	require.NoError(t, err)

	require.Equal(t, "created promotions pull request", promoteCommandMsgProd)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, promoteBranchName, group, "prod", app, tag)
	statusCommandMsgProd, err := StatusCommand(ctx, providerTypeString, path, password)
	require.NoError(t, err)

	require.Equal(t, "status check has succeed", statusCommandMsgProd)

	repoProd := testGetRepository(t, path)
	revProd := testGetRepositoryHeadRevision(t, repoProd)

	testMergePR(t, ctx, providerType, url, password, promoteBranchName, revProd)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, defaultBranch, group, "prod", app, tag)

	repoMergedProd := testGetRepository(t, path)
	revMergedProd := testGetRepositoryHeadRevision(t, repoMergedProd)

	testSetStatus(t, ctx, providerType, revMergedProd, group, "prod", url, password, true)
}
