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

	newCommandMsgDev, err := NewCommand(ctx, path, password, group, app, tag)
	require.NoError(t, err)

	require.Equal(t, "created promotions pull request", newCommandMsgDev)

	path = testCloneRepositoryAndValidateTag(t, url, username, password, defaultBranch, group, "dev", app, tag)

	repo := testGetRepository(t, path)
	rev := testGetRevision(t, repo)

	testSetAzureDevOpsStatus(t, rev, group, "dev", url, password, true)

	promoteCommandMsgQa, err := PromoteCommand(ctx, path, password)
	require.NoError(t, err)

	require.Equal(t, "created promotions pull request", promoteCommandMsgQa)

	promoteBranchName := fmt.Sprintf("promote/%s-%s", group, app)
	path = testCloneRepositoryAndValidateTag(t, url, username, password, promoteBranchName, group, "qa", app, tag)
	statusCommandMsgQa, err := StatusCommand(ctx, path, password)
	require.NoError(t, err)

	require.Equal(t, "status check has succeed", statusCommandMsgQa)
}
