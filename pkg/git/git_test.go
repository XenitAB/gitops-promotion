package git

import (
	"fmt"
	"os"
	"testing"

	scgit "github.com/fluxcd/source-controller/pkg/git"
	git2go "github.com/libgit2/git2go/v31"
	"github.com/stretchr/testify/require"
)

func TestPRState(t *testing.T) {
	cases := []struct {
		state PRState
	}{
		{
			state: PRState{
				Env:   "ENV_TESTING",
				Group: "GROUP_TESTING",
				App:   "APP_TESTING",
				Tag:   "TAG_TESTING",
				Sha:   "SHA_TESTING",
			},
		},
	}

	for _, c := range cases {
		title := c.state.Title()
		branchName := c.state.BranchName()
		description, err := c.state.Description()
		require.NoError(t, err)
		require.Contains(t, title, fmt.Sprintf("Promote %s", c.state.Group))
		require.Contains(t, branchName, fmt.Sprintf("%s%s-%s", PromoteBranchPrefix, c.state.Group, c.state.App))
		require.Contains(t, description, "<!-- metadata = ")
		require.Contains(t, description, " -->")
		require.Contains(t, description, c.state.Env)
		require.Contains(t, description, c.state.Group)
		require.Contains(t, description, c.state.App)
		require.Contains(t, description, c.state.Tag)
		require.Contains(t, description, c.state.Sha)

		parsedState, err := parsePrState(description)
		require.NoError(t, err)

		require.Equal(t, c.state.Env, parsedState.Env)
		require.Equal(t, c.state.Group, parsedState.Group)
		require.Equal(t, c.state.App, parsedState.App)
		require.Equal(t, c.state.Tag, parsedState.Tag)
		require.Equal(t, c.state.Sha, parsedState.Sha)
	}
}

func TestNewPR(t *testing.T) {
	cases := []struct {
		id          *int
		title       *string
		description *string
		prState     *PRState
		expectedErr string
	}{
		{
			id:          toIntPtr(1),
			title:       toStringPtr("testTitle"),
			description: toStringPtr("test description"),
			prState:     nil,
			expectedErr: "",
		},
		{
			id:          toIntPtr(1),
			title:       toStringPtr("testTitle"),
			description: toStringPtr("test description"),
			prState:     &PRState{},
			expectedErr: "",
		},
		{
			id:          toIntPtr(1),
			title:       toStringPtr("testTitle"),
			description: toStringPtr("test description"),
			prState: &PRState{
				Env:   "",
				Group: "",
				App:   "",
				Tag:   "",
				Sha:   "",
			},
			expectedErr: "",
		},
		{
			id:          toIntPtr(1),
			title:       toStringPtr("testTitle"),
			description: nil,
			prState:     nil,
			expectedErr: "",
		},
		{
			id:          nil,
			title:       toStringPtr("testTitle"),
			description: toStringPtr("test description"),
			prState:     nil,
			expectedErr: "id can't be empty",
		},
		{
			id:          toIntPtr(1),
			title:       nil,
			description: toStringPtr("test description"),
			prState:     nil,
			expectedErr: "title can't be empty",
		},
	}

	for _, c := range cases {
		_, err := newPR(c.id, c.title, c.description, c.prState)
		if err != nil && c.expectedErr == "" {
			t.Errorf("Expected err to be nil: %q", err)
		}

		if err == nil && c.expectedErr != "" {
			t.Errorf("Expected err not to be nil")
		}

		if err != nil && c.expectedErr != "" {
			if err.Error() != c.expectedErr {
				t.Errorf("Expected err to be '%q' but received: %q", c.expectedErr, err.Error())
			}
		}
	}
}

func toStringPtr(s string) *string {
	return &s
}

func toIntPtr(i int) *int {
	return &i
}

func testGetEnvOrSkip(t *testing.T, key string) string {
	t.Helper()

	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Skipping test since environment variable %q is not set", key)
	}

	return value
}

func testCloneRepository(t *testing.T, url, username, password, path string) {
	t.Helper()

	auth, err := testBasicAuthMethod(username, password)
	require.NoError(t, err)

	_, err = git2go.Clone(url, path, &git2go.CloneOptions{
		FetchOptions: &git2go.FetchOptions{
			DownloadTags: git2go.DownloadTagsNone,
			RemoteCallbacks: git2go.RemoteCallbacks{
				CredentialsCallback: auth.CredCallback,
			},
		},
		CheckoutBranch: DefaultBranch,
	})
	require.NoError(t, err)
}

func testBasicAuthMethod(username, password string) (*scgit.Auth, error) {
	credCallback := func(url string, usernameFromURL string, allowedTypes git2go.CredType) (*git2go.Cred, error) {
		cred, err := git2go.NewCredUserpassPlaintext(username, password)
		if err != nil {
			return nil, err
		}
		return cred, nil
	}

	return &scgit.Auth{CredCallback: credCallback}, nil
}
