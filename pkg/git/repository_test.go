package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAzureDevopsRepository(t *testing.T) {
	azdoPAT := getEnvOrSkip(t, "AZDO_PAT")
	azdoURL := getEnvOrSkip(t, "AZDO_URL")

	path := t.TempDir()
	ctx := context.Background()

	err := cloneRepository(azdoURL, "gitops-promotion", azdoPAT, path)
	require.NoError(t, err)

	repo, err := LoadRepository(ctx, path, ProviderTypeAzdo, azdoPAT)
	require.NoError(t, err)

	rootDir := repo.GetRootDir()
	require.Equal(t, path, rootDir)

	err = repo.CreateBranch("testing", false)
	require.NoError(t, err)

	err = repo.CreateBranch("testing-force", true)
	require.NoError(t, err)

	defaultBranchCommitID, err := repo.GetCurrentCommit()
	require.NoError(t, err)

	testingBranchCommitID, err := repo.GetLastCommitForBranch("testing")
	require.NoError(t, err)

	require.Equal(t, testingBranchCommitID, defaultBranchCommitID)
}
