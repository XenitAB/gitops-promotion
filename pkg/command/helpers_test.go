package command

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	scgit "github.com/fluxcd/source-controller/pkg/git"
	git2go "github.com/libgit2/git2go/v31"
	azdo "github.com/microsoft/azure-devops-go-api/azuredevops"
	azdogit "github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"github.com/stretchr/testify/require"
	giturls "github.com/whilp/git-urls"
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
	timeSleep := time.Duration(backoff/2+rand.Intn(backoff)) * time.Millisecond
	time.Sleep(timeSleep)
}

func testCloneRepository(t *testing.T, url, username, password, path, branchName string) {
	t.Helper()

	auth := testBasicAuthMethod(t, username, password)

	_, err := git2go.Clone(url, path, &git2go.CloneOptions{
		FetchOptions: &git2go.FetchOptions{
			DownloadTags: git2go.DownloadTagsNone,
			RemoteCallbacks: git2go.RemoteCallbacks{
				CredentialsCallback: auth.CredCallback,
			},
		},
		CheckoutBranch: branchName,
	})
	require.NoError(t, err)
}

func testBasicAuthMethod(t *testing.T, username, password string) *scgit.Auth {
	t.Helper()

	credCallback := func(url string, usernameFromURL string, allowedTypes git2go.CredType) (*git2go.Cred, error) {
		cred, err := git2go.NewCredUserpassPlaintext(username, password)
		if err != nil {
			return nil, err
		}
		return cred, nil
	}

	return &scgit.Auth{CredCallback: credCallback}
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

func testSetAzureDevOpsStatus(t *testing.T, revision, group, env, url, token string, succeeded bool) {
	t.Helper()

	genre := "fluxcd"
	description := fmt.Sprintf("testing-%s-%s-%s", group, env, revision)
	name := fmt.Sprintf("kind/%s-%s", group, env)
	orgURL, project, repository := testGetAzureDevOpsStrings(t, url)
	azdoClient := testGetAzureDevopsClient(t, orgURL, token)

	state := &azdogit.GitStatusStateValues.Succeeded
	if !succeeded {
		state = &azdogit.GitStatusStateValues.Failed
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	createArgs := azdogit.CreateCommitStatusArgs{
		Project:      &project,
		RepositoryId: &repository,
		CommitId:     &revision,
		GitCommitStatusToCreate: &azdogit.GitStatus{
			Description: &description,
			State:       state,
			Context: &azdogit.GitStatusContext{
				Genre: &genre,
				Name:  &name,
			},
		},
	}

	_, err := azdoClient.CreateCommitStatus(ctx, createArgs)
	require.NoError(t, err)
}

func testMergeAzureDevOpsPR(t *testing.T, ctx context.Context, url, token, branch, revision string) {
	t.Helper()

	provider, err := git.NewAzdoGITProvider(ctx, url, token)
	require.NoError(t, err)

	pr, err := provider.GetPRWithBranch(ctx, branch, "main")
	require.NoError(t, err)

	err = provider.MergePR(ctx, pr.ID, revision)
	require.NoError(t, err)
}

func testGetAzureDevOpsStrings(t *testing.T, s string) (string, string, string) {
	t.Helper()

	u, err := giturls.Parse(s)
	require.NoError(t, err)

	scheme := u.Scheme
	if u.Scheme == "ssh" {
		scheme = "https"
	}

	id := strings.TrimLeft(u.Path, "/")
	id = strings.TrimSuffix(id, ".git")
	host := fmt.Sprintf("%s://%s", scheme, u.Host)

	comp := strings.Split(id, "/")
	if len(comp) != 4 {
		require.NoError(t, fmt.Errorf("invalid repository id %q", id))
	}

	organization := comp[0]
	project := comp[1]
	repository := comp[3]

	orgURL := fmt.Sprintf("%v/%v", host, organization)

	return orgURL, project, repository
}

func testGetAzureDevopsClient(t *testing.T, orgURL, token string) *azdogit.ClientImpl {
	t.Helper()

	connection := azdo.NewPatConnection(orgURL, token)
	client := connection.GetClientByUrl(orgURL)
	return &azdogit.ClientImpl{
		Client: *client,
	}
}
