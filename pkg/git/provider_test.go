package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewGitProvider(t *testing.T) {
	cases := []struct {
		testDescription      string
		providerString       string
		expectedProviderType ProviderType
		remoteURL            string
		token                string
		expectedError        string
	}{
		{
			testDescription:      "azdo provider returns error",
			providerString:       "azdo",
			expectedProviderType: ProviderTypeAzdo,
			remoteURL:            "https://dev.azure.com/organization/project/_git/repository",
			token:                "fake",
			expectedError:        "TF400813: The user '' is not authorized to access this resource.",
		},
		{
			testDescription:      "github provider returns error",
			providerString:       "github",
			expectedProviderType: ProviderTypeGitHub,
			remoteURL:            "https://github.com/organization/repository",
			token:                "fake",
			expectedError:        "unable to authenticate using token",
		},
		{
			testDescription:      "fake provider returns error",
			providerString:       "fake",
			expectedProviderType: "",
			remoteURL:            "",
			token:                "",
			expectedError:        "Unknown provider selected: fake",
		},
	}

	for i, c := range cases {
		t.Logf("Test iteration %d: %s", i, c.testDescription)

		ctx := context.Background()

		providerType, err := StringToProviderType(c.providerString)
		if err != nil {
			require.EqualError(t, err, c.expectedError)
		}

		if err == nil {
			require.Equal(t, c.expectedProviderType, providerType)
			_, err := NewGitProvider(ctx, providerType, c.remoteURL, c.token)
			if c.expectedError != "" {
				require.EqualError(t, err, c.expectedError)
			}
		}
	}
}
