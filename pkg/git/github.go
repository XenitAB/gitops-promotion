package git

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/google/go-github/v40/github"
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

	host, id, err := ParseGitAddress(remoteURL)
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

	return &GitHubGITProvider{
		authClient: tc,
		client:     client,
		owner:      owner,
		repo:       repo,
	}, nil
}

// CreatePR ...
//nolint:gocognit //temporary
func (g *GitHubGITProvider) CreatePR(ctx context.Context, branchName string, auto bool, state *PRState) (int, error) {
	sourceName := branchName
	targetName := DefaultBranch
	title := state.Title()
	description, err := state.Description()
	if err != nil {
		return 0, err
	}

	listOpts := &github.PullRequestListOptions{
		State: "open",
		Base:  targetName,
	}

	openPrs, _, err := g.client.PullRequests.List(ctx, g.owner, g.repo, listOpts)
	if err != nil {
		return 0, err
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
		if err == nil {
			log.Printf("Created new PR #%d merging %s -> %s\n", pr.GetNumber(), sourceName, targetName)
		}
	case 1:
		pr = (prsOnBranch)[0]
		pr.Title = &title
		pr.Body = &description
		pr.Base.Ref = &targetName
		pr, _, err = g.client.PullRequests.Edit(ctx, g.owner, g.repo, *pr.Number, pr)
		if err == nil {
			log.Printf("Updated PR #%d merging %s -> %s\n", pr.GetNumber(), sourceName, targetName)
		}
	default:
		return 0, fmt.Errorf("received more than one PRs when listing: %d", len(prsOnBranch))
	}

	if err != nil {
		return 0, err
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
		if err == nil {
			log.Printf("Auto-merge activated for PR #%d\n", pr.GetNumber())
		} else {
			log.Printf("Failed to activate auto-merge for PR %d: %v", pr.GetNumber(), err)
			if strings.Contains(err.Error(), "Can't enable auto-merge") {
				err = fmt.Errorf("could not set auto-merge on PR #%d (check auto-merge setting and required checks)", pr.GetNumber())
			}
		}
	}
	return pr.GetNumber(), err
}

func (g *GitHubGITProvider) GetStatus(ctx context.Context, sha string, group string, env string) (CommitStatus, error) {
	opts := &github.ListOptions{PerPage: 50}
	statuses, _, err := g.client.Repositories.ListStatuses(ctx, g.owner, g.repo, sha, opts)
	if err != nil {
		return CommitStatus{}, err
	}
	var displays = make([]string, len(statuses))
	for i := range statuses {
		s := statuses[i]
		displays = append(displays, fmt.Sprintf("%s: %s (%s)", *s.Context, *s.State, *s.Description))
	}
	log.Printf("Considering statuses %v\n", displays)

	name := fmt.Sprintf("%s-%s", group, env)
	for _, s := range statuses {
		comp := strings.Split(*s.Context, "/")
		if len(comp) != 2 {
			return CommitStatus{}, fmt.Errorf("status context in wrong format: %q", *s.Context)
		}
		if comp[1] == name {
			return CommitStatus{
				Succeeded: *s.State == "success",
			}, nil
		}
	}
	return CommitStatus{}, fmt.Errorf("no status found for sha %q", sha)
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

	var result *github.PullRequestMergeResult
	var res *github.Response
	err := retry.Do(
		func() error {
			var err error
			result, res, err = g.client.PullRequests.Merge(ctx, g.owner, g.repo, id, "", opts)
			if err != nil && res.StatusCode == 405 {
				updateOpts := &github.PullRequestBranchUpdateOptions{}
				_, _, innerErr := g.client.PullRequests.UpdateBranch(ctx, g.owner, g.repo, id, updateOpts)
				if innerErr != nil {
					return innerErr
				}
			}
			return err
		},
		retry.Attempts(5),
		retry.LastErrorOnly(true),
	)
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

	var prs []*github.PullRequest
	err := retry.Do(
		func() error {
			openPrs, _, err := g.client.PullRequests.List(ctx, g.owner, g.repo, listOpts)
			if err != nil {
				return err
			}
			for _, pr := range openPrs {
				if source == *pr.Head.Ref {
					prs = append(prs, pr)
				}
			}
			if len(prs) != 1 {
				return fmt.Errorf("no PR found for branches %q-%q", source, target)
			}
			return nil
		},
		retry.Attempts(5),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		return PullRequest{}, err
	}

	pr := prs[0]

	return NewPullRequest(pr.Number, pr.Title, pr.Body)
}

// nolint:gocognit // ignore
func (g *GitHubGITProvider) GetPRThatCausedCommit(ctx context.Context, sha string) (PullRequest, error) {
	listOpts := &github.PullRequestListOptions{
		State: "closed",
	}

	var prs []*github.PullRequest
	err := retry.Do(
		func() error {
			closedPrs, _, err := g.client.PullRequests.List(ctx, g.owner, g.repo, listOpts)
			if err != nil {
				return err
			}
			for _, pr := range closedPrs {
				if pr == nil {
					continue
				}
				// The SHA will be nil if the PR is closed without being merged
				if pr.MergeCommitSHA == nil {
					continue
				}
				if sha == *pr.MergeCommitSHA {
					prs = append(prs, pr)
				}
			}
			if len(prs) != 1 {
				return fmt.Errorf("no PR found for sha: %s", sha)
			}
			return nil
		},
		retry.Attempts(5),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		return PullRequest{}, err
	}
	pr := prs[0]

	return NewPullRequest(pr.Number, pr.Title, pr.Body)
}
