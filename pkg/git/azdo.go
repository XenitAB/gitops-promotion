package git

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/webapi"
)

// AdoGITProvider ...
type AzdoGITProvider struct {
	client git.Client
	proj   string
	repo   string
}

// NewAdoGITProvider ...
func NewAzdoGITProvider(ctx context.Context, remoteURL, token string) (*AzdoGITProvider, error) {
	host, id, err := parseGitAddress(remoteURL)
	if err != nil {
		return nil, err
	}

	var org string
	var proj string
	var repo string
	if host == "https://dev.azure.com" {
		comp := strings.Split(id, "/")
		if len(comp) != 4 {
			return nil, fmt.Errorf("invalid repository id %q", id)
		}
		org = comp[0]
		proj = comp[1]
		repo = comp[3]
	} else {
		comp := strings.Split(id, "/")
		if len(comp) != 3 {
			return nil, fmt.Errorf("invalid repository id %q", id)
		}
		proj = comp[0]
		repo = comp[2]

		u, err := url.Parse(host)
		if err != nil {
			return nil, err
		}
		comp = strings.Split(u.Hostname(), ".")
		org = comp[0]
		host = "https://dev.azure.com"
	}

	connection := azuredevops.NewPatConnection(fmt.Sprintf("%s/%s", host, org), token)
	client, err := git.NewClient(ctx, connection)
	if err != nil {
		return nil, err
	}
	return &AzdoGITProvider{
		client: client,
		proj:   proj,
		repo:   repo,
	}, nil
}

// CreatePR ...
func (g *AzdoGITProvider) CreatePR(ctx context.Context, branchName string, auto bool, state *PRState) error {
	sourceRefName := fmt.Sprintf("refs/heads/%s", branchName)
	targetRefName := fmt.Sprintf("refs/heads/%s", DefaultBranch)
	title := state.Title()
	description, err := state.Description()
	if err != nil {
		return err
	}

	// Update PR if it already exists
	getArgs := git.GetPullRequestsArgs{
		Project:      &g.proj,
		RepositoryId: &g.repo,
		SearchCriteria: &git.GitPullRequestSearchCriteria{
			SourceRefName: &sourceRefName,
			TargetRefName: &targetRefName,
		},
	}
	prs, err := g.client.GetPullRequests(ctx, getArgs)
	if err == nil && len(*prs) > 0 {
		pr := (*prs)[0]
		var autoCompleteSetBy *webapi.IdentityRef
		if auto {
			autoCompleteSetBy = pr.CreatedBy
		}
		updatePR := git.GitPullRequest{
			Title:             &title,
			Description:       &description,
			AutoCompleteSetBy: autoCompleteSetBy,
		}
		updateArgs := git.UpdatePullRequestArgs{
			Project:                &g.proj,
			RepositoryId:           &g.repo,
			PullRequestId:          pr.PullRequestId,
			GitPullRequestToUpdate: &updatePR,
		}
		_, err = g.client.UpdatePullRequest(ctx, updateArgs)
		return err
	}

	// Create new PR
	deleteSourceBranch := true
	createArgs := git.CreatePullRequestArgs{
		Project:      &g.proj,
		RepositoryId: &g.repo,
		GitPullRequestToCreate: &git.GitPullRequest{
			Title:         &title,
			Description:   &description,
			SourceRefName: &sourceRefName,
			TargetRefName: &targetRefName,
			CompletionOptions: &git.GitPullRequestCompletionOptions{
				DeleteSourceBranch: &deleteSourceBranch,
			},
		},
	}
	pr, err := g.client.CreatePullRequest(ctx, createArgs)
	if err != nil {
		return err
	}
	if !auto {
		return nil
	}

	// This update is done to set auto merge. The reason this is not
	// done when creating is because there is no reasonable way to
	// get the identity ref other than from the response.
	updatePR := git.GitPullRequest{
		AutoCompleteSetBy: pr.CreatedBy,
	}
	updateArgs := git.UpdatePullRequestArgs{
		Project:                &g.proj,
		RepositoryId:           &g.repo,
		PullRequestId:          pr.PullRequestId,
		GitPullRequestToUpdate: &updatePR,
	}
	_, err = g.client.UpdatePullRequest(ctx, updateArgs)
	return err
}

func (g *AzdoGITProvider) GetStatus(ctx context.Context, sha string, group string, env string) (Status, error) {
	args := git.GetStatusesArgs{
		Project:      &g.proj,
		RepositoryId: &g.repo,
		CommitId:     &sha,
	}
	statuses, err := g.client.GetStatuses(ctx, args)
	if err != nil {
		return Status{}, err
	}
	genre := "fluxcd"
	name := fmt.Sprintf("%s-%s", group, env)
	for i := range *statuses {
		s := (*statuses)[i]
		comp := strings.Split(*s.Context.Name, "/")
		if len(comp) != 2 {
			return Status{}, fmt.Errorf("status name in wrong format: %q", *s.Context.Name)
		}
		if *s.Context.Genre == genre && comp[1] == name {
			return Status{
				Succeeded: *s.State == git.GitStatusStateValues.Succeeded,
			}, nil
		}
	}
	return Status{}, fmt.Errorf("no status found for sha %q", sha)
}

func (g *AzdoGITProvider) MergePR(ctx context.Context, id int, sha string) error {
	args := git.UpdatePullRequestArgs{
		Project:       &g.proj,
		RepositoryId:  &g.repo,
		PullRequestId: &id,
		GitPullRequestToUpdate: &git.GitPullRequest{
			Status: &git.PullRequestStatusValues.Completed,
			LastMergeSourceCommit: &git.GitCommitRef{
				CommitId: &sha,
			},
		},
	}
	_, err := g.client.UpdatePullRequest(ctx, args)
	return err
}

func (g *AzdoGITProvider) GetPRWithBranch(ctx context.Context, source, target string) (PullRequest, error) {
	sourceRefName := fmt.Sprintf("refs/heads/%s", source)
	targetRefName := fmt.Sprintf("refs/heads/%s", target)
	args := git.GetPullRequestsArgs{
		Project:      &g.proj,
		RepositoryId: &g.repo,
		SearchCriteria: &git.GitPullRequestSearchCriteria{
			SourceRefName: &sourceRefName,
			TargetRefName: &targetRefName,
		},
	}
	prs, err := g.client.GetPullRequests(ctx, args)
	if err != nil {
		return PullRequest{}, err
	}
	if len(*prs) == 0 {
		return PullRequest{}, fmt.Errorf("no PR found for branches %q-%q", source, target)
	}

	pr := (*prs)[0]

	result, err := newPR(pr.PullRequestId, pr.Title, pr.Description, nil)
	if err != nil {
		return PullRequest{}, err
	}

	return result, nil
}

func (g *AzdoGITProvider) GetPRThatCausedCommit(ctx context.Context, sha string) (PullRequest, error) {
	args := git.GetPullRequestQueryArgs{
		Project:      &g.proj,
		RepositoryId: &g.repo,
		Queries: &git.GitPullRequestQuery{
			Queries: &[]git.GitPullRequestQueryInput{
				{
					Items: &[]string{sha},
					Type:  &git.GitPullRequestQueryTypeValues.LastMergeCommit,
				},
			},
		},
	}
	query, err := g.client.GetPullRequestQuery(ctx, args)
	if err != nil {
		return PullRequest{}, err
	}
	results := *query.Results
	if len(results[0]) == 0 {
		return PullRequest{}, fmt.Errorf("no PR found for commit %q", sha)
	}
	pr := results[0][sha][0]

	result, err := newPR(pr.PullRequestId, pr.Title, pr.Description, nil)
	if err != nil {
		return PullRequest{}, err
	}

	return result, nil
}
