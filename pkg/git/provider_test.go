package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewGitProvider(t *testing.T) {
	cases := []struct {
		providerString       string
		expectedProviderType ProviderType
		remoteURL            string
		token                string
		expectedError        string
	}{
		{
			providerString:       "azdo",
			expectedProviderType: ProviderTypeAzdo,
			remoteURL:            "https://dev.azure.com/organization/project/_git/repository",
			token:                "fake",
			expectedError:        "",
		},
		{
			providerString:       "fake",
			expectedProviderType: ProviderTypeUnknown,
			remoteURL:            "",
			token:                "",
			expectedError:        "unknown provider type: unknown",
		},
	}

	for _, c := range cases {
		ctx := context.Background()

		providerType := StringToProviderType(c.providerString)
		require.Equal(t, c.expectedProviderType, providerType)

		_, err := NewGitProvider(ctx, providerType, c.remoteURL, c.token)
		if c.expectedError != "" {
			require.Error(t, err, c.expectedError)
		}
	}
}
