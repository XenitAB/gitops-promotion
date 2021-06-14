package git

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	git2go "github.com/libgit2/git2go/v31"
)

// Repository represents a local git repository.
type Repository struct {
	gitRepository *git2go.Repository
	gitProvider   GitProvider
	token         string
}

// LoadRepository loads a local git repository.
func LoadRepository(ctx context.Context, path string, providerType ProviderType, token string) (*Repository, error) {
	localRepo, err := git2go.OpenRepository(path)
	if err != nil {
		return &Repository{}, fmt.Errorf("could not open repository: %w", err)
	}

	remote, err := localRepo.Remotes.Lookup(DefaultRemote)
	if err != nil {
		return nil, fmt.Errorf("could not get remote: %w", err)
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

func (g *Repository) GetLastCommitForBranch(branchName string) (*git2go.Oid, error) {
	branch, err := g.gitRepository.LookupBranch(branchName, git2go.BranchAll)
	if err != nil {
		return nil, err
	}

	return branch.Target(), nil
}

func (g *Repository) GetCurrentCommit() (*git2go.Oid, error) {
	head, err := g.gitRepository.Head()
	if err != nil {
		return nil, err
	}
	return head.Target(), nil
}

// CreateCommit creates a commit in the specfied branch.
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
	sha, err := g.gitRepository.CreateCommit(fmt.Sprintf("refs/heads/%s", branchName), signature, signature, message, tree, commitTarget)
	if err != nil {
		return nil, err
	}

	return sha, nil
}

// Push pushes the defined ref to remote.
func (g *Repository) Push(branchName string) error {
	remote, err := g.gitRepository.Remotes.Lookup(DefaultRemote)
	if err != nil {
		return fmt.Errorf("could not find remote %q: %w", DefaultRemote, err)
	}

	callback := git2go.RemoteCallbacks{
		CredentialsCallback: func(url string, usernameFromURL string, allowedTypes git2go.CredType) (*git2go.Cred, error) {
			cred, err := git2go.NewCredentialUserpassPlaintext(DefaultUsername, g.token)
			if err != nil {
				return nil, err
			}
			return cred, nil
		},
	}
	branches := []string{fmt.Sprintf("+refs/heads/%s", branchName)}
	err = remote.Push(branches, &git2go.PushOptions{RemoteCallbacks: callback})
	if err != nil {
		return fmt.Errorf("failed pushing branches %s: %w", branches, err)
	}
	return nil
}

func (g *Repository) CreatePR(ctx context.Context, branchName string, auto bool, state *PRState) error {
	return g.gitProvider.CreatePR(ctx, branchName, auto, state)
}

func (g *Repository) GetStatus(ctx context.Context, sha, group, env string) (Status, error) {
	return g.gitProvider.GetStatus(ctx, sha, group, env)
}

func (g *Repository) MergePR(ctx context.Context, id int, sha string) error {
	return g.gitProvider.MergePR(ctx, id, sha)
}

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

func (g *Repository) GetPRForCurrentBranch(ctx context.Context) (PullRequest, error) {
	branchName, err := g.GetBranchName()
	if err != nil {
		return PullRequest{}, err
	}
	pr, err := g.gitProvider.GetPRWithBranch(ctx, branchName, DefaultBranch)
	if err != nil {
		return PullRequest{}, err
	}
	state, err := parsePrState(pr.Description)
	if err != nil {
		return PullRequest{}, err
	}
	pr.State = state
	return pr, nil
}

func (g *Repository) GetPRThatCausedCurrentCommit(ctx context.Context) (PullRequest, error) {
	head, err := g.gitRepository.Head()
	if err != nil {
		return PullRequest{}, err
	}
	pr, err := g.gitProvider.GetPRThatCausedCommit(ctx, head.Target().String())
	if err != nil {
		return PullRequest{}, err
	}
	state, err := parsePrState(pr.Description)
	if err != nil {
		return PullRequest{}, err
	}
	pr.State = state
	return pr, err
}
