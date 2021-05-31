package git

import (
	"os"
	"testing"

	scgit "github.com/fluxcd/source-controller/pkg/git"
	git2go "github.com/libgit2/git2go/v31"
)

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

func getEnvOrSkip(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Skipping test since environment variable %q is not set", key)
	}

	return value
}

func cloneRepository(url, username, password, path string) error {
	auth, err := basicAuthMethod(username, password)
	if err != nil {
		return err
	}

	_, err = git2go.Clone(url, path, &git2go.CloneOptions{
		FetchOptions: &git2go.FetchOptions{
			DownloadTags: git2go.DownloadTagsNone,
			RemoteCallbacks: git2go.RemoteCallbacks{
				CredentialsCallback: auth.CredCallback,
			},
		},
		CheckoutBranch: DefaultBranch,
	})
	if err != nil {
		return err
	}

	return nil
}

func basicAuthMethod(username, password string) (*scgit.Auth, error) {
	credCallback := func(url string, usernameFromURL string, allowedTypes git2go.CredType) (*git2go.Cred, error) {
		cred, err := git2go.NewCredUserpassPlaintext(username, password)
		if err != nil {
			return nil, err
		}
		return cred, nil
	}

	return &scgit.Auth{CredCallback: credCallback}, nil
}
