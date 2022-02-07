package git

import (
	"encoding/json"
	"fmt"
	"strings"
)

type PRType string

const (
	PRTypePromote PRType = "promote"
	PRTypeFeature PRType = "feature"
)

type PullRequest struct {
	ID          int
	Title       string
	Description string
	State       *PRState
}

func NewPullRequest(id *int, title *string, description *string) (PullRequest, error) {
	if id == nil {
		return PullRequest{}, fmt.Errorf("id can't be empty")
	}
	if title == nil {
		return PullRequest{}, fmt.Errorf("title can't be empty")
	}
	d := ""
	if description != nil {
		d = *description
	}
	state, _, err := NewPRState(d)
	if err != nil {
		return PullRequest{}, err
	}
	return PullRequest{
		ID:          *id,
		Title:       *title,
		Description: d,
		State:       state,
	}, nil
}

type PRState struct {
	Group string `json:"group"`
	App   string `json:"app"`
	Tag   string `json:"tag"`
	Env   string `json:"env"`
	Sha   string `json:"sha"`
	Type  PRType `json:"type"`
}

// NewPRState takes the content of a pull rquest description and coverts
// it to a PRState. No error will be returned if the description does not
// contain state metadata, but the bool value will be false.
func NewPRState(description string) (*PRState, bool, error) {
	// Check if the body contains state data. If it does not it should return nil.
	comp := strings.Split(description, " -->")
	if len(comp) < 2 {
		return nil, false, nil
	}
	comp = strings.Split(comp[0], "<!-- metadata = ")
	if len(comp) < 2 {
		return nil, false, nil
	}

	// Parse the state json data
	out := comp[1]
	prState := &PRState{}
	err := json.Unmarshal([]byte(out), prState)
	if err != nil {
		return nil, false, err
	}
	return prState, true, nil
}

func (p *PRState) GetPRType() PRType {
	// Needed for backwards compatibility
	if p.Type == "" {
		return PRTypePromote
	}
	return p.Type
}

func (p *PRState) BranchName(includeEnv bool) string {
	comps := []string{string(p.GetPRType())}
	if includeEnv {
		comps = append(comps, p.Env)
	}
	name := fmt.Sprintf("%s-%s", p.Group, p.App)
	if p.GetPRType() == PRTypeFeature {
		name = fmt.Sprintf("%s-%s", name, p.Tag)
	}
	comps = append(comps, name)
	return strings.Join(comps, "/")
}

func (p *PRState) Title() string {
	switch p.GetPRType() {
	case PRTypePromote:
		return fmt.Sprintf("Promote %s/%s version %s to environment %s", p.Group, p.App, p.Tag, p.Env)
	case PRTypeFeature:
		return fmt.Sprintf("Review %s/%s feature %s in environment %s", p.Group, p.App, p.Tag, p.Env)
	default:
		return ""
	}
}

func (p *PRState) Description() (string, error) {
	jsonString, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	description := fmt.Sprintf(`<!-- metadata = %s -->
	ENV: %s
	APP: %s
	TAG: %s`, string(jsonString), p.Env, p.App, p.Tag)
	return description, nil
}
