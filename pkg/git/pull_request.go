package git

import (
	"encoding/json"
	"fmt"
	"strings"
)

type PullRequest struct {
	ID          int
	Title       string
	Description string
	State       PRState
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
	state, err := NewPRState(d)
	if err != nil {
		return PullRequest{}, nil
	}
	return PullRequest{
		ID:          *id,
		Title:       *title,
		Description: d,
		State:       state,
	}, nil
}

type PRType string

const (
	PRTypePromote PRType = "promote"
)

type PRState struct {
	Env   string `json:"env"`
	Group string `json:"group"`
	App   string `json:"app"`
	Tag   string `json:"tag"`
	Sha   string `json:"sha"`
	Type  PRType `json:"type"`
}

func NewPRState(body string) (PRState, error) {
	comp := strings.Split(body, " -->")
	if len(comp) < 2 {
		return PRState{}, fmt.Errorf("invalid metadata: %q", body)
	}
	comp = strings.Split(comp[0], "<!-- metadata = ")
	if len(comp) < 2 {
		return PRState{}, fmt.Errorf("invalid metadata: %q", body)
	}
	out := comp[1]
	prState := PRState{}
	err := json.Unmarshal([]byte(out), &prState)
	if err != nil {
		return PRState{}, err
	}
	return prState, nil
}

func (p *PRState) GetPRType() PRType {
	// Needed for backwards compatibility
	if p.Type == "" {
		return PRTypePromote
	}
	return p.Type
}

func (p *PRState) BranchName(includeEnv bool) string {
	if includeEnv {
		return fmt.Sprintf("%s/%s/%s-%s", p.GetPRType(), p.Env, p.Group, p.App)
	}
	return fmt.Sprintf("%s/%s-%s", p.GetPRType(), p.Group, p.App)
}

func (p *PRState) Title() string {
	switch p.GetPRType() {
	case PRTypePromote:
		return fmt.Sprintf("Promote %s/%s version %s to environment %s", p.Group, p.App, p.Tag, p.Env)
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
