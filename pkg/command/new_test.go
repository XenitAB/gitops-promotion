// package command

// import (
// 	"context"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/require"
// )

// func TestNewCommandAzdo(t *testing.T) {
// 	azdoPAT := testGetEnvOrSkip(t, "AZDO_PAT")
// 	// azdoURL := testGetEnvOrSkip(t, "AZDO_URL")

// 	cases := []struct {
// 		path            string
// 		token           string
// 		group           string
// 		app             string
// 		expectedMessage string
// 		expectedError   string
// 	}{
// 		{
// 			path:            t.TempDir(),
// 			token:           azdoPAT,
// 			group:           "testgroup",
// 			app:             "testapp",
// 			expectedMessage: "created promotions pull request",
// 			expectedError:   "",
// 		},
// 	}

// 	for _, c := range cases {
// 		now := time.Now()
// 		tag := now.Format("20060102150405")
// 		ctx := context.Background()

// 		msg, err := NewCommand(ctx, c.path, c.token, c.group, c.app, tag)
// 		if c.expectedError != "" {
// 			require.Error(t, err)
// 		} else {
// 			require.NoError(t, err)
// 			require.Equal(t, c.expectedMessage, msg)
// 		}
// 	}
// }
