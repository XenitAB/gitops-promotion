package git

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v37/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// GitHubGITProvider ...
type GitHubGITProvider struct {
	authClient *http.Client
	client     *github.Client
	owner      string
	repo       string
}

// NewGitHubGITProvider ...
func NewGitHubGITProvider(ctx context.Context, remoteURL, token string) (*GitHubGITProvider, error) {
	if remoteURL == "" {
		return nil, fmt.Errorf("remoteURL empty")
	}

	if token == "" {
		return nil, fmt.Errorf("token empty")
	}

	host, id, err := parseGitAddress(remoteURL)
	if err != nil {
		return nil, err
	}

	if host != "https://github.com" {
		return nil, fmt.Errorf("host does not start with https://github.com: %s", host)
	}

	comp := strings.Split(id, "/")
	if len(comp) != 2 {
		return nil, fmt.Errorf("invalid repository id %q", id)
	}

	owner := comp[0]
	repo := comp[1]

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	_, _, err = client.Repositories.List(ctx, "", nil)
	if err != nil {
		var githubError *github.ErrorResponse
		if errors.As(err, &githubError) {
			if githubError.Response.StatusCode == 401 {
				return nil, fmt.Errorf("unable to authenticate using token")
			}
		}

		return nil, err
	}

	return &GitHubGITProvider{
		authClient: tc,
		client:     client,
		owner:      owner,
		repo:       repo,
	}, nil
}

// CreatePR ...
func (g *GitHubGITProvider) CreatePR(ctx context.Context, branchName string, auto bool, state *PRState) error {
	sourceName := branchName
	targetName := DefaultBranch
	title := state.Title()
	description, err := state.Description()
	if err != nil {
		return err
	}

	listOpts := &github.PullRequestListOptions{
		State: "open",
		Base:  targetName,
	}

	openPrs, _, err := g.client.PullRequests.List(ctx, g.owner, g.repo, listOpts)
	if err != nil {
		return err
	}

	var prsOnBranch []*github.PullRequest
	for _, pr := range openPrs {
		if sourceName == *pr.Head.Ref {
			prsOnBranch = append(prsOnBranch, pr)
		}
	}

	var pr *github.PullRequest
	switch len(prsOnBranch) {
	case 0:
		createOpts := &github.NewPullRequest{
			Title:               &title,
			Body:                &description,
			Head:                &sourceName,
			Base:                &targetName,
			MaintainerCanModify: github.Bool(true),
		}
		pr, _, err = g.client.PullRequests.Create(ctx, g.owner, g.repo, createOpts)
	case 1:
		pr = (prsOnBranch)[0]
		pr.Title = &title
		pr.Body = &description
		pr.Base.Ref = &targetName
		pr, _, err = g.client.PullRequests.Edit(ctx, g.owner, g.repo, *pr.Number, pr)
	default:
		return fmt.Errorf("received more than one PRs when listing: %d", len(prsOnBranch))
	}

	if err != nil {
		return err
	}

	if auto != (pr.GetAutoMerge() != nil) {
		client := githubv4.NewClient(g.authClient)
		var mutation struct {
			EnablePullRequestAutoMerge struct {
				PullRequest struct {
					ID githubv4.ID
				}
			} `graphql:"enablePullRequestAutoMerge(input: $input)"`
		}
		input := githubv4.EnablePullRequestAutoMergeInput{
			PullRequestID: pr.GetNodeID(),
		}
		err = client.Mutate(ctx, &mutation, input, nil)
		if err != nil && strings.Contains(err.Error(), "not in the correct state") {
			err = g.MergePR(ctx, int(pr.GetNumber()), *pr.GetHead().SHA)
		}
	}
	return err
}

func (g *GitHubGITProvider) GetStatus(ctx context.Context, sha string, group string, env string) (Status, error) {
	opts := &github.ListOptions{PerPage: 50}
	statuses, _, err := g.client.Repositories.ListStatuses(ctx, g.owner, g.repo, sha, opts)
	if err != nil {
		return Status{}, err
	}

	name := fmt.Sprintf("%s-%s", group, env)
	for _, s := range statuses {
		comp := strings.Split(*s.Context, "/")
		if len(comp) != 2 {
			return Status{}, fmt.Errorf("status context in wrong format: %q", *s.Context)
		}
		if comp[1] == name {
			return Status{
				Succeeded: *s.State == "success",
			}, nil
		}
	}

	return Status{}, fmt.Errorf("no status found for sha %q", sha)
}

func (g *GitHubGITProvider) SetStatus(ctx context.Context, sha string, group string, env string, succeeded bool) error {
	description := fmt.Sprintf("%s-%s-%s", group, env, sha)
	name := fmt.Sprintf("kind/%s-%s", group, env)

	state := "success"
	if !succeeded {
		state = "failure"
	}

	status := &github.RepoStatus{
		State:       &state,
		Context:     &name,
		Description: &description,
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, _, err := g.client.Repositories.CreateStatus(ctx, g.owner, g.repo, sha, status)
	return err
}

func (g *GitHubGITProvider) MergePR(ctx context.Context, id int, sha string) error {
	opts := &github.PullRequestOptions{
		SHA: sha,
	}

	result, res, err := g.client.PullRequests.Merge(ctx, g.owner, g.repo, id, "", opts)
	if err != nil {
		return err
	}

	mergeSucceeded := *result.Merged

	if !mergeSucceeded {
		body, err := ioutil.ReadAll(res.Response.Body)
		if err != nil {
			return err
		}

		defer func() {
			err := res.Response.Body.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "MergePR - unable to close body: %v", err)
			}
		}()

		return fmt.Errorf("PR with ID %d was not merged: %s", id, body)
	}

	return nil
}

func (g *GitHubGITProvider) GetPRWithBranch(ctx context.Context, source, target string) (PullRequest, error) {
	listOpts := &github.PullRequestListOptions{
		State: "open",
		Base:  target,
	}

	openPrs, _, err := g.client.PullRequests.List(ctx, g.owner, g.repo, listOpts)
	if err != nil {
		return PullRequest{}, err
	}

	var prs []*github.PullRequest
	for _, pr := range openPrs {
		if source == *pr.Head.Ref {
			prs = append(prs, pr)
		}
	}

	if len(prs) == 0 {
		return PullRequest{}, fmt.Errorf("no PR found for branches %q-%q", source, target)
	}

	pr := prs[0]

	return newPR(pr.Number, pr.Title, pr.Body, nil)
}

func (g *GitHubGITProvider) GetPRThatCausedCommit(ctx context.Context, sha string) (PullRequest, error) {
	listOpts := &github.PullRequestListOptions{
		State: "closed",
	}

	closedPrs, _, err := g.client.PullRequests.List(ctx, g.owner, g.repo, listOpts)
	if err != nil {
		return PullRequest{}, err
	}

	var prs []*github.PullRequest
	for _, pr := range closedPrs {
		if sha == *pr.MergeCommitSHA {
			prs = append(prs, pr)
		}
	}

	if len(prs) == 0 {
		return PullRequest{}, fmt.Errorf("no PR found for sha: %s", sha)
	}

	pr := prs[0]

	return newPR(pr.Number, pr.Title, pr.Body, nil)
}
