package git

import "testing"

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
