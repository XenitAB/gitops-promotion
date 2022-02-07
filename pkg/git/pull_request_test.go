package git

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPullRequest(t *testing.T) {
	cases := []struct {
		id          *int
		title       *string
		description *string
		expectedErr string
	}{
		{
			id:          toIntPtr(1),
			title:       toStringPtr("testTitle"),
			description: toStringPtr("test description"),
			expectedErr: "",
		},
		{
			id:          toIntPtr(1),
			title:       toStringPtr("testTitle"),
			description: toStringPtr("test description"),
			expectedErr: "",
		},
		{
			id:          toIntPtr(1),
			title:       toStringPtr("testTitle"),
			description: toStringPtr("test description"),
			expectedErr: "",
		},
		{
			id:          toIntPtr(1),
			title:       toStringPtr("testTitle"),
			description: nil,
			expectedErr: "",
		},
		{
			id:          nil,
			title:       toStringPtr("testTitle"),
			description: toStringPtr("test description"),
			expectedErr: "id can't be empty",
		},
		{
			id:          toIntPtr(1),
			title:       nil,
			description: toStringPtr("test description"),
			expectedErr: "title can't be empty",
		},
	}
	for _, c := range cases {
		_, err := NewPullRequest(c.id, c.title, c.description)
		if c.expectedErr != "" {
			require.EqualError(t, err, c.expectedErr)
		} else {
			require.NoError(t, err)
		}
	}
}

func toStringPtr(s string) *string {
	return &s
}

func toIntPtr(i int) *int {
	return &i
}

func TestPRStateValid(t *testing.T) {
	json := `{"group":"g","app":"a","tag":"t","env":"e","sha":"s","type":"promote"}`
	description := fmt.Sprintf("<!-- metadata = %s -->\n\tENV: e\n\tAPP: a\n\tTAG: t", json)
	state, ok, err := NewPRState(description)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "g", state.Group)
	require.Equal(t, "a", state.App)
	require.Equal(t, "t", state.Tag)
	require.Equal(t, "e", state.Env)
	require.Equal(t, "s", state.Sha)
	require.Equal(t, PRTypePromote, state.Type)
	genDescription, err := state.Description()
	require.NoError(t, err)
	require.Equal(t, description, genDescription)
}

func TestPRStateInvalid(t *testing.T) {
	description := "<!-- metadata = {{ asdasd } -->"
	_, ok, err := NewPRState(description)
	require.Error(t, err)
	require.False(t, ok)
}

func TestPRStateNone(t *testing.T) {
	description := "Some other data"
	state, ok, err := NewPRState(description)
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, state)
}

func TestPRStateBranchName(t *testing.T) {
	cases := []struct {
		name               string
		state              PRState
		includeEnv         bool
		expectedBranchName string
	}{
		{
			name: "promote no env",
			state: PRState{
				Group: "group",
				App:   "app",
				Tag:   "tag",
				Env:   "dev",
				Type:  PRTypePromote,
			},
			includeEnv:         false,
			expectedBranchName: "promote/group-app",
		},
		{
			name: "promote include env",
			state: PRState{
				Group: "group",
				App:   "app",
				Tag:   "tag",
				Env:   "dev",
				Type:  PRTypePromote,
			},
			includeEnv:         true,
			expectedBranchName: "promote/dev/group-app",
		},
		{
			name: "feature no env",
			state: PRState{
				Group: "group",
				App:   "app",
				Tag:   "tag",
				Env:   "dev",
				Type:  PRTypeFeature,
			},
			includeEnv:         false,
			expectedBranchName: "feature/group-app-tag",
		},
		{
			name: "feature include env",
			state: PRState{
				Group: "group",
				App:   "app",
				Tag:   "tag",
				Env:   "dev",
				Type:  PRTypeFeature,
			},
			includeEnv:         true,
			expectedBranchName: "feature/dev/group-app-tag",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.expectedBranchName, c.state.BranchName(c.includeEnv))
		})
	}
}
