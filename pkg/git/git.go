package git

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	git "github.com/libgit2/git2go/v31"
	git2go "github.com/libgit2/git2go/v31"
)

const (
	DefaultUsername = "git"
	DefaultRemote   = "origin"
	DefaultBranch   = "main"
)

type CommitStatus struct {
	Succeeded bool
}

// Repository represents a local git repository.
type Repository struct {
	gitRepository *git2go.Repository
	gitProvider   GitProvider
	token         string
}

// LoadRepository loads a local git repository.
func LoadRepository(ctx context.Context, path string, providerTypeString string, token string) (*Repository, error) {
	localRepo, err := git2go.OpenRepository(path)
	if err != nil {
		return &Repository{}, fmt.Errorf("could not open repository: %w", err)
	}
	remote, err := localRepo.Remotes.Lookup(DefaultRemote)
	if err != nil {
		return nil, fmt.Errorf("could not get remote: %w", err)
	}
	providerType, err := StringToProviderType(providerTypeString)
	if err != nil {
		return nil, fmt.Errorf("could not get provider type: %w", err)
	}
	provider, err := NewGitProvider(ctx, providerType, remote.Url(), token)
	if err != nil {
		return nil, fmt.Errorf("could not create git provider: %w", err)
	}
	return &Repository{
		gitRepository: localRepo,
		gitProvider:   provider,
		token:         token,
	}, nil
}

// FetchDefaultBranch updates DefaultBranch with new commits from DefaultRemote.
func (g *Repository) FetchBranch(branchName string) (*git.Oid, error) {
	remote, err := g.gitRepository.Remotes.Lookup(DefaultRemote)
	if err != nil {
		return nil, fmt.Errorf("could not find remote %q: %w", DefaultRemote, err)
	}
	err = remote.Fetch(
		[]string{branchName},
		&git2go.FetchOptions{
			RemoteCallbacks: credentialsCallback(DefaultUsername, g.token),
		},
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	sha, err := g.GetLastCommitForBranch(fmt.Sprintf("%s/%s", DefaultRemote, DefaultBranch))
	if err != nil {
		return nil, fmt.Errorf("fetch failed to lookup head sha: %w", err)
	}
	return sha, nil
}

// GetRootDir returns the file path to the repository.
func (g *Repository) GetRootDir() string {
	p := g.gitRepository.Path()
	rp := filepath.Clean(filepath.Join(p, ".."))
	return rp
}

// CreateBranch creates a branch.
func (g *Repository) CreateBranch(branchName string, force bool) error {
	branch, err := g.gitRepository.LookupBranch(branchName, git2go.BranchLocal)
	if err == nil {
		err = branch.Delete()
		if err != nil {
			return fmt.Errorf("could not delete existing branch %q: %w", branchName, err)
		}
	}
	head, err := g.gitRepository.Head()
	if err != nil {
		return err
	}
	headCommit, err := g.gitRepository.LookupCommit(head.Target())
	if err != nil {
		return err
	}
	_, err = g.gitRepository.CreateBranch(branchName, headCommit, force)
	if err != nil {
		return err
	}
	return nil
}

// GetLastCommitForBranch returns the latest commit id for the branch.
func (g *Repository) GetLastCommitForBranch(branchName string) (*git2go.Oid, error) {
	branch, err := g.gitRepository.LookupBranch(branchName, git2go.BranchRemote)
	if err != nil {
		return nil, err
	}
	return branch.Target(), nil
}

// GetCurrentCommit returns the commmit id for HEAD.
func (g *Repository) GetCurrentCommit() (*git2go.Oid, error) {
	head, err := g.gitRepository.Head()
	if err != nil {
		return nil, err
	}
	return head.Target(), nil
}

// CreateCommit creates a commit in the specfied branch with the current changes.
func (g *Repository) CreateCommit(branchName, message string) (*git2go.Oid, error) {
	// TODO change to some bot name, probably break out in to config
	signature := &git2go.Signature{
		Name:  "gitops-promotion",
		Email: "gitops-promotion@xenit.se",
		When:  time.Now(),
	}
	idx, err := g.gitRepository.Index()
	if err != nil {
		return nil, err
	}
	err = idx.AddAll([]string{}, git2go.IndexAddDefault, nil)
	if err != nil {
		return nil, err
	}
	treeId, err := idx.WriteTree()
	if err != nil {
		return nil, err
	}
	err = idx.Write()
	if err != nil {
		return nil, err
	}
	tree, err := g.gitRepository.LookupTree(treeId)
	if err != nil {
		return nil, err
	}
	branch, err := g.gitRepository.LookupBranch(branchName, git2go.BranchLocal)
	if err != nil {
		return nil, err
	}
	commitTarget, err := g.gitRepository.LookupCommit(branch.Target())
	if err != nil {
		return nil, err
	}
	refName := fmt.Sprintf("refs/heads/%s", branchName)
	sha, err := g.gitRepository.CreateCommit(refName, signature, signature, message, tree, commitTarget)
	if err != nil {
		return nil, err
	}
	log.Printf("Created commit %s on %s with message '%s'\n", sha, refName, message)
	return sha, nil
}

// Push pushes the given branch to the remote.
func (g *Repository) Push(branchName string, force bool) error {
	remote, err := g.gitRepository.Remotes.Lookup(DefaultRemote)
	if err != nil {
		return fmt.Errorf("could not find remote %q: %w", DefaultRemote, err)
	}

	forceFlag := "+"
	if !force {
		forceFlag = ""
	}

	branches := []string{fmt.Sprintf("%srefs/heads/%s", forceFlag, branchName)}
	err = remote.Push(branches, &git2go.PushOptions{RemoteCallbacks: credentialsCallback(DefaultUsername, g.token)})
	if err != nil {
		return fmt.Errorf("failed pushing branches %s: %w", branches, err)
	}
	log.Printf("Pushed branch %s to remote\n", branches[0])
	return nil
}

// CreatePR creates a PR for the branch. It assumes that the branch has been pushed.
func (g *Repository) CreatePR(ctx context.Context, branchName string, auto bool, state *PRState) (int, error) {
	return g.gitProvider.CreatePR(ctx, branchName, auto, state)
}

// GetStatus returns the status for the give commit.
func (g *Repository) GetStatus(ctx context.Context, sha, group, env string) (CommitStatus, error) {
	return g.gitProvider.GetStatus(ctx, sha, group, env)
}

// MergePR merges the given PR.
func (g *Repository) MergePR(ctx context.Context, id int, sha string) error {
	return g.gitProvider.MergePR(ctx, id, sha)
}

// GetBranchName returns the branch name of for HEAD.
func (g *Repository) GetBranchName() (string, error) {
	head, err := g.gitRepository.Head()
	if err != nil {
		return "", err
	}
	branchName, err := head.Branch().Name()
	if err != nil {
		return "", err
	}
	return branchName, nil
}

// GetPRForCurrentBranch returns any active PR for the current branch.
func (g *Repository) GetPRForCurrentBranch(ctx context.Context) (PullRequest, error) {
	branchName, err := g.GetBranchName()
	if err != nil {
		return PullRequest{}, err
	}
	pr, err := g.gitProvider.GetPRWithBranch(ctx, branchName, DefaultBranch)
	if err != nil {
		return PullRequest{}, err
	}
	return pr, nil
}

// GetPRThatCausedCurrentCommit finds the merged PR with resulted in the current commit.
func (g *Repository) GetPRThatCausedCurrentCommit(ctx context.Context) (PullRequest, error) {
	head, err := g.gitRepository.Head()
	if err != nil {
		return PullRequest{}, err
	}
	pr, err := g.gitProvider.GetPRThatCausedCommit(ctx, head.Target().String())
	if err != nil {
		return PullRequest{}, err
	}
	return pr, err
}

func Clone(url, username, password, path, branchName string) error {
	_, err := git2go.Clone(url, path, &git2go.CloneOptions{
		FetchOptions: &git2go.FetchOptions{
			DownloadTags:    git2go.DownloadTagsNone,
			RemoteCallbacks: credentialsCallback(username, password),
		},
		CheckoutBranch: branchName,
	})
	return err
}

func credentialsCallback(username, password string) git2go.RemoteCallbacks {
	return git2go.RemoteCallbacks{
		CredentialsCallback: func(url string, usernameFromURL string, allowedTypes git2go.CredentialType) (*git2go.Credential, error) {
			cred, err := git2go.NewCredentialUserpassPlaintext(username, password)
			if err != nil {
				return nil, err
			}
			return cred, nil
		},
	}
}
