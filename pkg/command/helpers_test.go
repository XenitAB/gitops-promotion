package command

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	git2go "github.com/libgit2/git2go/v31"
	"github.com/stretchr/testify/require"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

func testGetEnvOrSkip(t *testing.T, key string) string {
	t.Helper()

	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Skipping test since environment variable %q is not set", key)
	}

	return value
}

func testCloneRepositoryAndValidateTag(t *testing.T, url, username, password, branchName, group, env, app, tag string) string {
	t.Helper()

	manifestPath := fmt.Sprintf("%s/%s/%s.yaml", group, env, app)
	for i := 1; i < 10; i++ {
		path := t.TempDir()
		testCloneRepository(t, url, username, password, path, branchName)
		fileName := fmt.Sprintf("%s/%s", path, manifestPath)

		content, err := os.ReadFile(fileName)
		require.NoError(t, err)

		if strings.Contains(string(content), tag) {
			return path
		}

		testSleepBackoff(t, i)
	}

	t.Fatalf("Was not able to pull the latest commit where %q contained tag: %s", manifestPath, tag)
	return ""
}

func testSleepBackoff(t *testing.T, i int) {
	t.Helper()

	backoff := i * 200

	// #nosec
	timeSleep := time.Duration(backoff/2+rand.Intn(backoff)) * time.Millisecond

	time.Sleep(timeSleep)
}

func testCloneRepository(t *testing.T, url, username, password, path, branchName string) {
	t.Helper()

	err := git.Clone(url, username, password, path, branchName)
	require.NoError(t, err)
}

func testGetRepository(t *testing.T, path string) *git2go.Repository {
	t.Helper()

	localRepo, err := git2go.OpenRepository(path)
	require.NoError(t, err)

	return localRepo
}

func testGetRepositoryHeadRevision(t *testing.T, repo *git2go.Repository) string {
	t.Helper()

	head, err := repo.Head()
	require.NoError(t, err)

	rev := head.Target().String()

	return rev
}

func testSetStatus(
	t *testing.T,
	ctx context.Context,
	providerType git.ProviderType,
	revision,
	group,
	env,
	url,
	token string,
	succeeded bool,
) {
	t.Helper()

	repo, err := git.NewGitProvider(ctx, providerType, url, token)
	require.NoError(t, err)

	err = repo.SetStatus(ctx, revision, group, env, succeeded)
	require.NoError(t, err)
}

func testMergePR(t *testing.T, ctx context.Context, providerType git.ProviderType, url, token, branch, revision string) {
	t.Helper()

	provider, err := git.NewGitProvider(ctx, providerType, url, token)
	require.NoError(t, err)

	pr, err := provider.GetPRWithBranch(ctx, branch, "main")
	require.NoError(t, err)

	err = provider.MergePR(ctx, pr.ID, revision)
	require.NoError(t, err)
}
