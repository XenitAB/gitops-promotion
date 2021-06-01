package git

import (
	"context"
	"os"
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
			expectedError:        "TF400813: The user '' is not authorized to access this resource.",
		},
		{
			providerString:       "fake",
			expectedProviderType: ProviderTypeUnknown,
			remoteURL:            "",
			token:                "",
			expectedError:        "unknown provider type: unknown",
		},
	}

	azdoPAT, azdoURL, testAzdo := testNewGitProviderAzdo(t)
	if testAzdo {
		azdoCase := struct {
			providerString       string
			expectedProviderType ProviderType
			remoteURL            string
			token                string
			expectedError        string
		}{
			providerString:       "azdo",
			expectedProviderType: ProviderTypeAzdo,
			remoteURL:            azdoURL,
			token:                azdoPAT,
			expectedError:        "",
		}
		cases = append(cases, azdoCase)
	}

	for _, c := range cases {
		ctx := context.Background()

		providerType := StringToProviderType(c.providerString)
		require.Equal(t, c.expectedProviderType, providerType)

		_, err := NewGitProvider(ctx, providerType, c.remoteURL, c.token)
		if c.expectedError != "" {
			require.EqualError(t, err, c.expectedError)
		}
	}
}

func testNewGitProviderAzdo(t *testing.T) (string, string, bool) {
	t.Helper()

	azdoPAT := os.Getenv("AZDO_PAT")
	azdoURL := os.Getenv("AZDO_URL")

	if azdoPAT != "" && azdoURL != "" {
		return azdoPAT, azdoURL, true
	}

	return "", "", false
}
