package git

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGitAddress(t *testing.T) {
	cases := []struct {
		input         string
		expectedHost  string
		expectedID    string
		expectedError string
	}{
		{
			input:         "https://dev.azure.com/organization/project/_git/repository",
			expectedHost:  "https://dev.azure.com",
			expectedID:    "organization/project/_git/repository",
			expectedError: "",
		},
		{
			input:         "https://user@dev.azure.com/organization/project/_git/repository",
			expectedHost:  "https://dev.azure.com",
			expectedID:    "organization/project/_git/repository",
			expectedError: "",
		},
		{
			input:         "ssh://dev.azure.com/organization/project/_git/repository",
			expectedHost:  "https://dev.azure.com",
			expectedID:    "organization/project/_git/repository",
			expectedError: "",
		},
		{
			input:         "https://dev.azure.com/organization/project/_git/repository.git",
			expectedHost:  "https://dev.azure.com",
			expectedID:    "organization/project/_git/repository",
			expectedError: "",
		},
		{
			input:         "/tmp/organization/project/_git/repository.git",
			expectedHost:  "file://",
			expectedID:    "tmp/organization/project/_git/repository",
			expectedError: "",
		},
	}

	for _, c := range cases {
		host, id, err := parseGitAddress(c.input)
		if c.expectedError != "" {
			require.EqualError(t, err, c.expectedError)
		}

		if c.expectedError == "" {
			require.Equal(t, c.expectedHost, host)
			require.Equal(t, c.expectedID, id)
		}
	}
}
