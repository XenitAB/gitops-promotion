package command

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/xenitab/gitops-promotion/pkg/git"
)

func TestUpdateImageTag(t *testing.T) {
	cases := []struct {
		state               git.PRState
		before              string
		after               string
		expectedErrContains string
		expectedMatch       bool
	}{
		{
			state: git.PRState{
				Env:   "dev",
				Group: "team1",
				App:   "app1",
				Tag:   "v1.0.1",
			},
			before: `random: test
image: app1:v1.0.0 # {"$imagepolicy": "team1:app1"}
why: true
`,
			after: `random: test
image: app1:v1.0.1 # {"$imagepolicy": "team1:app1"}
why: true
`,
			expectedErrContains: "",
			expectedMatch:       true,
		},
		{
			state: git.PRState{
				Env:   "dev",
				Group: "team1",
				App:   "app1",
				Tag:   "v1.0.1",
			},
			before: `random: test
tag: v1.0.0 # {"$imagepolicy": "team1:app1:tag"}
why: true
`,
			after: `random: test
tag: v1.0.1 # {"$imagepolicy": "team1:app1:tag"}
why: true
`,
			expectedErrContains: "",
			expectedMatch:       true,
		},
		{
			state: git.PRState{
				Env:   "dev",
				Group: "team1",
				App:   "app1",
				Tag:   "v1.0.1",
			},
			before: `random: test
tag: "1234" # {"$imagepolicy": "team1:app1:tag"}
why: true
`,
			after: `random: test
tag: "v1.0.1" # {"$imagepolicy": "team1:app1:tag"}
why: true
`,
			expectedErrContains: "",
			expectedMatch:       true,
		},
		{
			state: git.PRState{
				Env:   "dev",
				Group: "team1",
				App:   "app1",
				Tag:   "v1.0.1",
			},
			before: `random: test
tag: 1234 # {"$imagepolicy": "team1:app1:tag"}
why: true
`,
			after: `random: test
tag: v1.0.1 # {"$imagepolicy": "team1:app1:tag"}
why: true
`,
			expectedErrContains: "",
			expectedMatch:       false, // This is a know issue: https://github.com/XenitAB/gitops-promotion/issues/32
		},
	}

	for i, c := range cases {
		dir, err := os.MkdirTemp("", fmt.Sprintf("%d", i))
		if err != nil && c.expectedErrContains == "" {
			t.Errorf("Expected err to be nil: %q", err)
		}

		defer os.RemoveAll(dir)

		testFile := fmt.Sprintf("%s/%s.yaml", dir, c.state.App)
		err = os.WriteFile(testFile, []byte(c.before), 0666)
		if err != nil && c.expectedErrContains == "" {
			t.Errorf("Expected err to be nil: %q", err)
		}

		err = updateImageTag(dir, c.state.App, c.state.Group, c.state.Tag)
		if err == nil && c.expectedErrContains != "" {
			t.Errorf("Expected err not to be nil")
		}

		if err != nil && c.expectedErrContains != "" {
			if !strings.Contains(err.Error(), c.expectedErrContains) {
				t.Errorf("Expected err to contain '%q' but received: %q", c.expectedErrContains, err.Error())
			}
		}

		if err == nil && c.expectedErrContains == "" {
			result, err := os.ReadFile(testFile)
			if err != nil && c.expectedErrContains == "" {
				t.Errorf("Expected err to be nil: %q", err)
			}

			if string(result) != c.after && c.expectedMatch {
				t.Errorf("\nExpected:\n%s\n\nReceived:\n%s\n", c.after, string(result))
			}
		}

	}
}
