package git

import (
	"encoding/json"
	"fmt"
	"strings"
)

const DefaultUsername = "git"
const DefaultRemote = "origin"
const DefaultBranch = "main"

type Status struct {
	Succeeded bool
}

type PullRequest struct {
	ID          int
	Title       string
	Description string
	State       PRState
}

type PRState struct {
	Env   string `json:"env"`
	Group string `json:"group"`
	App   string `json:"app"`
	Tag   string `json:"tag"`
	Sha   string `json:"sha"`
}

func (p PRState) Title() string {
	return fmt.Sprintf("Promote %s/%s in %s to %s", p.Group, p.App, p.Env, p.Tag)
}

func (p PRState) Description() (string, error) {
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

func (p PRState) BranchName() string {
	return fmt.Sprintf("promote/%s-%s", p.Group, p.App)
}

func parsePrState(body string) (PRState, error) {
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

func newPR(id *int, title *string, description *string, state *PRState) (PullRequest, error) {
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

	s := PRState{}
	if state != nil {
		s = *state
	}

	return PullRequest{
		ID:          *id,
		Title:       *title,
		Description: d,
		State:       s,
	}, nil
}
