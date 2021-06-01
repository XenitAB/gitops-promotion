package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAzureDevops(t *testing.T) {
	azdoPAT := testGetEnvOrSkip(t, "AZDO_PAT")
	azdoURL := testGetEnvOrSkip(t, "AZDO_URL")

	repo := testAzdoRepository(t, azdoURL, azdoPAT)
	testBranch(t, repo, "testing")
	branchName, timestamp := testCommit(t, repo, false, "testing")
	testPushAndMerge(t, repo, branchName)

	testCleanupAzdoRepository(t, azdoURL, azdoPAT, timestamp)
}

func testAzdoRepository(t *testing.T, url, password string) *Repository {
	t.Helper()

	path := t.TempDir()
	ctx := context.Background()

	testCloneRepository(t, url, "gitops-promotion", password, path)
	repo, err := LoadRepository(ctx, path, ProviderTypeAzdo, password)
	require.NoError(t, err)

	rootDir := repo.GetRootDir()
	require.Equal(t, path, rootDir)

	return repo
}

func testBranch(t *testing.T, repo *Repository, branchName string) {
	t.Helper()

	err := repo.CreateBranch(branchName, false)
	require.NoError(t, err)

	forceBranchName := fmt.Sprintf("%s-force", branchName)
	err = repo.CreateBranch(forceBranchName, true)
	require.NoError(t, err)

	defaultBranchCommitID, err := repo.GetCurrentCommit()
	require.NoError(t, err)

	testingBranchCommitID, err := repo.GetLastCommitForBranch(branchName)
	require.NoError(t, err)

	require.Equal(t, testingBranchCommitID, defaultBranchCommitID)
}

func testCommit(t *testing.T, repo *Repository, cleanup bool, branchNamePrefix string) (string, string) {
	t.Helper()

	root := repo.GetRootDir()
	now := time.Now()
	timestamp := now.Format("20060102150405")
	testFileName := fmt.Sprintf("%s/test_%s.txt", root, timestamp)
	branchName := fmt.Sprintf("%s_%s", branchNamePrefix, timestamp)
	commitMessage := fmt.Sprintf("testing: %s", timestamp)

	testFilesGlob := fmt.Sprintf("%s/test_*.txt", root)
	testRemoveTemporaryTestFiles(t, testFilesGlob)

	if !cleanup {
		err := os.WriteFile(testFileName, []byte(timestamp), 0600)
		require.NoError(t, err)
	}

	err := repo.CreateBranch(branchName, false)
	require.NoError(t, err)

	commitID, err := repo.CreateCommit(branchName, commitMessage)
	require.NoError(t, err)

	latestBranchCommitID, err := repo.GetLastCommitForBranch(branchName)
	require.NoError(t, err)

	require.Equal(t, commitID, latestBranchCommitID)

	return branchName, timestamp
}

func testPushAndMerge(t *testing.T, repo *Repository, branchName string) {
	t.Helper()

	err := repo.Push(branchName)
	require.NoError(t, err)

	ctx := context.Background()

	state := PRState{
		Env:   "",
		Group: "GROUP_TESTING",
		App:   "APP_TESTING",
		Tag:   "TAG_TESTING",
		Sha:   "",
	}

	err = repo.CreatePR(ctx, branchName, true, state)
	require.NoError(t, err)
}

func testRemoveTemporaryTestFiles(t *testing.T, path string) {
	contents, err := filepath.Glob(path)
	if err != nil {
		return
	}
	for _, item := range contents {
		err = os.RemoveAll(item)
		if err != nil {
			return
		}
	}
}

func testCleanupAzdoRepository(t *testing.T, url string, password string, timestamp string) {
	t.Helper()

	var repo *Repository
	var testFileName string

	ok := false
	for i := 1; i < 5; i++ {
		repo = testAzdoRepository(t, url, password)
		root := repo.GetRootDir()
		testFileName = fmt.Sprintf("%s/test_%s.txt", root, timestamp)

		if _, err := os.Stat(testFileName); !errors.Is(err, os.ErrNotExist) {
			ok = true
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	if !ok {
		t.Fatalf("Was not able to pull the latest commit containing file: %s", testFileName)
	}

	cleanupBranchName, _ := testCommit(t, repo, true, "cleanup")
	testPushAndMerge(t, repo, cleanupBranchName)
}
