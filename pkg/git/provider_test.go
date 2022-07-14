package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewGitProvider(t *testing.T) {
	tests := []struct {
		name         string
		providerType ProviderType
		remoteURL    string
		token        string
		expectedErr  string
	}{
		{
			name:         "azdo provider returns error",
			providerType: ProviderTypeAzdo,
			remoteURL:    "https://dev.azure.com/organization/project/_git/repository",
			token:        "fake",
			expectedErr:  "TF400813: The user 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa' is not authorized to access this resource.",
		},
		{
			name:         "fake provider returns error",
			providerType: ProviderType("fake"),
			remoteURL:    "",
			token:        "",
			expectedErr:  "unknown provider type: fake",
		},
	}

	for i, tt := range tests {
		t.Logf("Test iteration %d: %s", i, tt.name)
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGitProvider(context.TODO(), tt.providerType, tt.remoteURL, tt.token)
			require.EqualError(t, err, tt.expectedErr)
		})
	}
}
