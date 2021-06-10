package command

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestE2EAzureDevOps(t *testing.T) {
	username := "gitops-promotion"
	password := testGetEnvOrSkip(t, "AZDO_PAT")
	url := testGetEnvOrSkip(t, "AZDO_URL")
	defaultBranch := "main"
	path := t.TempDir()

	testCloneRepository(t, url, username, password, path, defaultBranch)

	now := time.Now()
	tag := now.Format("20060102150405")
	group := "testgroup"
	app := "testapp"

	ctx := context.Background()

	// Test DEV
	newCommandMsgDev, err := NewCommand(ctx, path, password, group, app, tag)
	require.NoError(t, err)

	require.Equal(t, "created promotions pull request", newCommandMsgDev)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, defaultBranch, group, "dev", app, tag)

	repoDev := testGetRepository(t, path)
	revDev := testGetRepositoryHeadRevision(t, repoDev)

	testSetAzureDevOpsStatus(t, revDev, group, "dev", url, password, true)

	// Test QA
	promoteCommandMsgQa, err := PromoteCommand(ctx, path, password)
	require.NoError(t, err)

	require.Equal(t, "created promotions pull request", promoteCommandMsgQa)

	promoteBranchName := fmt.Sprintf("promote/%s-%s", group, app)
	path = testCloneRepositoryAndValidateTag(t, url, username, password, promoteBranchName, group, "qa", app, tag)
	statusCommandMsgQa, err := StatusCommand(ctx, path, password)
	require.NoError(t, err)

	require.Equal(t, "status check has succeed", statusCommandMsgQa)

	repoQa := testGetRepository(t, path)
	revQa := testGetRepositoryHeadRevision(t, repoQa)

	testMergeAzureDevOpsPR(t, ctx, url, password, promoteBranchName, revQa)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, defaultBranch, group, "qa", app, tag)

	repoMergedQa := testGetRepository(t, path)
	revMergedQa := testGetRepositoryHeadRevision(t, repoMergedQa)

	testSetAzureDevOpsStatus(t, revMergedQa, group, "qa", url, password, true)

	// Test PROD
	promoteCommandMsgProd, err := PromoteCommand(ctx, path, password)
	require.NoError(t, err)

	require.Equal(t, "created promotions pull request", promoteCommandMsgProd)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, promoteBranchName, group, "prod", app, tag)
	statusCommandMsgProd, err := StatusCommand(ctx, path, password)
	require.NoError(t, err)

	require.Equal(t, "status check has succeed", statusCommandMsgProd)

	repoProd := testGetRepository(t, path)
	revProd := testGetRepositoryHeadRevision(t, repoProd)

	testMergeAzureDevOpsPR(t, ctx, url, password, promoteBranchName, revProd)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, defaultBranch, group, "prod", app, tag)

	repoMergedProd := testGetRepository(t, path)
	revMergedProd := testGetRepositoryHeadRevision(t, repoMergedProd)

	testSetAzureDevOpsStatus(t, revMergedProd, group, "prod", url, password, true)
}
