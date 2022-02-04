package tests

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v40/github"
	"github.com/google/uuid"
	git2go "github.com/libgit2/git2go/v31"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/xenitab/gitops-promotion/pkg/git"
)

var remoteURL string = os.Getenv("GITHUB_URL")
var token string = os.Getenv("GITHUB_TOKEN")
var createdRepos []string

func randomBranchName(prefix string) string {
	return prefix + "-" + uuid.NewString()
}

func cloneTestRepoOnExistingBranch(ctx context.Context, branchName string) *git.Repository {
	providerTypeString := string(git.ProviderTypeGitHub)
	tmpDir, e := ioutil.TempDir("", "gitops-promotion")
	createdRepos = append(createdRepos, tmpDir)
	Expect(e).To(BeNil())
	e = clone(remoteURL, "pat", token, tmpDir, branchName)
	Expect(e).To(BeNil())
	repo, e := git.LoadRepository(ctx, tmpDir, providerTypeString, token)
	Expect(e).To(BeNil())
	return repo
}

func cloneTestRepoWithNewBranch(ctx context.Context, branchName string) *git.Repository {
	repo := cloneTestRepoOnExistingBranch(ctx, git.DefaultBranch)
	e := repo.CreateBranch(branchName, false)
	Expect(e).To(BeNil())
	return repo
}

func pushBranch(repo *git.Repository, branchName string) {
	e := repo.Push(branchName, true)
	Expect(e).To(BeNil())
}

func commitAFile(repo *git.Repository, branchName string) *git2go.Oid {
	fileName := fmt.Sprintf(
		"%s/%s.txt",
		filepath.Dir(strings.TrimRight(repo.GetRootDir(), "/")),
		branchName,
	)
	f, e := os.Create(fileName)
	Expect(e).To(BeNil())
	_, e = f.WriteString(fmt.Sprintln(time.Now()))
	Expect(e).To(BeNil())
	e = f.Close()
	Expect(e).To(BeNil())
	sha, e := repo.CreateCommit(branchName, fmt.Sprintln(time.Now()))
	Expect(e).To(BeNil())
	return sha
}

var _ = AfterSuite(func() {
	for _, path := range createdRepos {
		e := os.RemoveAll(path)
		Expect(e).To(BeNil())
	}
})

func TestNewGitHubGITProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GitHubProvider")
}

var _ = Describe("NewGitHubGITProvider", func() {
	var err error
	var ctx context.Context

	BeforeEach(func() {
		err = nil
		ctx = context.Background()
	})

	It("returns error when creating without url", func() {
		_, err = git.NewGitHubGITProvider(ctx, "", "foo")
		Expect(err).To(MatchError("remoteURL empty"))
	})

	It("returns error when creating without token", func() {
		_, err = git.NewGitHubGITProvider(ctx, "https://github.com/org/repo", "")
		Expect(err).To(MatchError("token empty"))
	})

	It("returns error when creating without github address", func() {
		_, err = git.NewGitHubGITProvider(ctx, "https://foo.bar/org/repo", "foo")
		Expect(err).To(MatchError("host does not start with https://github.com: https://foo.bar"))
	})

	It("is successfully created when creating with correct token", func() {
		var provider *git.GitHubGITProvider
		remoteURL := os.Getenv("GITHUB_URL")
		token := os.Getenv("GITHUB_TOKEN")

		if remoteURL == "" || token == "" {
			Skip("GITHUB_URL and/or GITHUB_TOKEN environment variables not set")
		}

		provider, err = git.NewGitHubGITProvider(ctx, remoteURL, token)
		Expect(err).To(BeNil())
		Expect(remoteURL).To(ContainSubstring(provider.owner))
		Expect(remoteURL).To(ContainSubstring(provider.repo))
	})
})

var _ = Describe("GitHubGITProvider CreatePR", func() {
	ctx := context.Background()
	provider, providerErr := git.NewGitHubGITProvider(ctx, remoteURL, token)

	BeforeEach(func() {
		if remoteURL == "" || token == "" {
			Skip("GITHUB_URL and/or GITHUB_TOKEN environment variables not set")
		}

		if providerErr != nil {
			Fail(fmt.Sprintf("Provider initialization failed: %s", providerErr))
		}
	})

	var prid int
	var err error
	var branchName string
	var auto bool
	state := &git.PRState{
		Env:   "dev",
		Group: "testgroup",
		App:   "testapp",
		Tag:   time.Now().Format("20060102150405"),
		Sha:   "",
	}

	JustBeforeEach(func() {
		prid, err = provider.CreatePR(ctx, branchName, auto, state)
	})

	When("Creating PR with empty values", func() {
		It("returns error", func() {
			var gitHubError *github.ErrorResponse
			ok := errors.As(err, &gitHubError)
			Expect(ok).To(Equal(true))
			body, bodyErr := ioutil.ReadAll(gitHubError.Response.Body)
			Expect(bodyErr).To(BeNil())
			bodyErr = gitHubError.Response.Body.Close()
			Expect(bodyErr).To(BeNil())

			Expect(string(body)).To(ContainSubstring("{\"resource\":\"PullRequest\",\"code\":\"missing_field\",\"field\":\"head\"}"))
		})
	})

	When("Creating PR with non-existing branchName", func() {
		BeforeEach(func() {
			branchName = randomBranchName("testing-does-not-exist")
		})

		It("returns error", func() {
			var gitHubError *github.ErrorResponse
			ok := errors.As(err, &gitHubError)
			Expect(ok).To(Equal(true))
			body, bodyErr := ioutil.ReadAll(gitHubError.Response.Body)
			Expect(bodyErr).To(BeNil())
			bodyErr = gitHubError.Response.Body.Close()
			Expect(bodyErr).To(BeNil())

			Expect(string(body)).To(ContainSubstring("{\"resource\":\"PullRequest\",\"field\":\"head\",\"code\":\"invalid\"}"))
		})
	})

	When("Creating PR on a new branch", func() {
		var repo *git.Repository

		BeforeEach(func() {
			branchName = randomBranchName("empty-branch")
			repo = cloneTestRepoWithNewBranch(ctx, branchName)
			commitAFile(repo, branchName)
			pushBranch(repo, branchName)
		})

		It("doesn't return an error", func() {
			Expect(err).To(BeNil())
		})

		It("returns the PR number", func() {
			Expect(prid).To(BeNumerically(">", 0))
		})
	})

	When("Creating PR on a branch that already has a PR", func() {
		var origPRId int
		var repo *git.Repository

		BeforeEach(func() {
			var e error
			branchName = randomBranchName("testing-create-pr")
			repo = cloneTestRepoWithNewBranch(ctx, branchName)
			commitAFile(repo, branchName)
			pushBranch(repo, branchName)

			origPRId, e = provider.CreatePR(ctx, branchName, false, state)
			Expect(e).To(BeNil())
		})

		It("doesn't return an error", func() {
			Expect(err).To(BeNil())
		})

		It("returns the original PRs id", func() {
			Expect(prid).To(Equal(origPRId))
		})
	})

	When("Creating/updating a PR with automerge", func() {
		var repo *git.Repository

		BeforeEach(func() {
			auto = true
			branchName = randomBranchName("with-automerge")
			repo = cloneTestRepoWithNewBranch(ctx, branchName)
			commitAFile(repo, branchName)
			pushBranch(repo, branchName)
		})

		It("returns an error saying auto-merge is not turned on", func() {
			Expect(err.Error()).To(ContainSubstring("could not set auto-merge"))
		})

		When("and the repository has branch protection requiring passing statuses", func() {
			BeforeEach(func() {
				_, _, err := provider.client.Repositories.UpdateBranchProtection(
					ctx,
					provider.owner,
					provider.repo,
					git.DefaultBranch,
					&github.ProtectionRequest{
						EnforceAdmins: true,
						RequiredStatusChecks: &github.RequiredStatusChecks{
							Strict:   true,
							Contexts: []string{},
						},
					})
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				_, _, err := provider.client.Repositories.UpdateBranchProtection(
					ctx,
					provider.owner,
					provider.repo,
					DefaultBranch,
					&github.ProtectionRequest{})
				Expect(err).To(BeNil())
			})

			It("doesn't return an error", func() {
				Expect(err).To(BeNil())
			})
		})
	})
})

var _ = Describe("GitHubGITProvider GetStatus", func() {
	ctx := context.Background()
	provider, providerErr := git.NewGitHubGITProvider(ctx, remoteURL, token)

	BeforeEach(func() {
		if remoteURL == "" || token == "" {
			Skip("GITHUB_URL and/or GITHUB_TOKEN environment variables not set")
		}

		if providerErr != nil {
			Fail(fmt.Sprintf("Provider initialization failed: %s", providerErr))
		}
	})

	var err error
	var status git.CommitStatus
	state := &git.PRState{
		Env:   "dev",
		Group: "testgroup",
		App:   "testapp",
		Tag:   time.Now().Format("20060102150405"),
		Sha:   "",
	}

	JustBeforeEach(func() {
		status, err = provider.GetStatus(ctx, state.Sha, state.Group, state.Env)
	})

	When("Getting status of empty sha", func() {
		It("return error", func() {
			var gitHubError *github.ErrorResponse
			ok := errors.As(err, &gitHubError)
			Expect(ok).To(Equal(true))
			body, bodyErr := ioutil.ReadAll(gitHubError.Response.Body)
			Expect(bodyErr).To(BeNil())
			bodyErr = gitHubError.Response.Body.Close()
			Expect(bodyErr).To(BeNil())

			Expect(string(body)).To(ContainSubstring("\"message\":\"Not Found\""))
			Expect(status.Succeeded).To(Equal(false))
		})
	})

	When("Getting status of existing sha without status", func() {
		var repo *git.Repository

		BeforeEach(func() {
			var e error
			repo = cloneTestRepoWithNewBranch(ctx, "ignored")
			commitAFile(repo, git.DefaultBranch)
			pushBranch(repo, git.DefaultBranch)

			sha, e := repo.GetCurrentCommit()
			Expect(e).To(BeNil())
			state.Sha = sha.String()
		})

		It("returns an error", func() {
			Expect(err.Error()).To(ContainSubstring("no status found for sha"))
			Expect(status.Succeeded).To(Equal(false))
		})

		When("and when a status is set to failure", func() {
			BeforeEach(func() {
				e := provider.SetStatus(ctx, state.Sha, state.Group, state.Env, false)
				Expect(e).To(BeNil())
			})

			It("reports failure", func() {
				Expect(err).To(BeNil())
				Expect(status.Succeeded).To(Equal(false))
			})
		})

		When("and when a status is set to success", func() {
			BeforeEach(func() {
				e := provider.SetStatus(ctx, state.Sha, state.Group, state.Env, true)
				Expect(e).To(BeNil())
			})

			It("reports succeeds", func() {
				Expect(err).To(BeNil())
				Expect(status.Succeeded).To(Equal(true))
			})
		})
	})
})

var _ = Describe("GitHubGITProvider MergePR", func() {
	ctx := context.Background()
	provider, providerErr := git.NewGitHubGITProvider(ctx, remoteURL, token)

	BeforeEach(func() {
		if remoteURL == "" || token == "" {
			Skip("GITHUB_URL and/or GITHUB_TOKEN environment variables not set")
		}

		if providerErr != nil {
			Fail(fmt.Sprintf("Provider initialization failed: %s", providerErr))
		}
	})

	var err error
	var prID int
	now := time.Now()
	state := &git.PRState{
		Env:   "dev",
		Group: "testgroup",
		App:   "testapp",
		Tag:   now.Format("20060102150405"),
		Sha:   "",
	}

	JustBeforeEach(func() {
		err = provider.MergePR(ctx, prID, state.Sha)
	})

	When("Merging PR with empty prID and SHA", func() {
		It("return error", func() {
			var gitHubError *github.ErrorResponse
			ok := errors.As(err, &gitHubError)
			Expect(ok).To(Equal(true))
			body, bodyErr := ioutil.ReadAll(gitHubError.Response.Body)
			Expect(bodyErr).To(BeNil())
			bodyErr = gitHubError.Response.Body.Close()
			Expect(bodyErr).To(BeNil())

			Expect(string(body)).To(ContainSubstring("\"message\":\"Not Found\""))
		})
	})

	When("Merging PR with existing prID and SHA", func() {
		BeforeEach(func() {
			branchName := randomBranchName("testing-merge-pr")
			repo1 := cloneTestRepoWithNewBranch(ctx, branchName)
			pushBranch(repo1, branchName)
			repo2 := cloneTestRepoOnExistingBranch(ctx, branchName)
			state.Sha = commitAFile(repo2, branchName).String()
			pushBranch(repo2, branchName)
			_, e := provider.CreatePR(ctx, branchName, false, state)
			Expect(e).To(BeNil())

			pr, e := provider.GetPRWithBranch(ctx, branchName, git.DefaultBranch)
			Expect(e).To(BeNil())

			prID = pr.ID
		})

		It("doesn't return an error", func() {
			Expect(err).To(BeNil())
		})
	})
})

var _ = Describe("GitHubGITProvider GetPRWithBranch", func() {
	ctx := context.Background()
	provider, providerErr := git.NewGitHubGITProvider(ctx, remoteURL, token)

	BeforeEach(func() {
		if remoteURL == "" || token == "" {
			Skip("GITHUB_URL and/or GITHUB_TOKEN environment variables not set")
		}

		if providerErr != nil {
			Fail(fmt.Sprintf("Provider initialization failed: %s", providerErr))
		}
	})

	var err error
	var branchName string
	var pr git.PullRequest
	now := time.Now()
	state := &git.PRState{
		Env:   "dev",
		Group: "testgroup",
		App:   "testapp",
		Tag:   now.Format("20060102150405"),
		Sha:   "",
	}

	JustBeforeEach(func() {
		pr, err = provider.GetPRWithBranch(ctx, branchName, git.DefaultBranch)
	})

	When("Getting PR with empty branchName", func() {
		It("returns an error", func() {
			Expect(err.Error()).To(ContainSubstring("no PR found for branches"))
		})
	})

	When("Getting PR with existing branchName", func() {
		var origPRId int
		BeforeEach(func() {
			branchName = randomBranchName("testing-get-pr-with-branch")

			var e error
			repo1 := cloneTestRepoWithNewBranch(ctx, branchName)
			pushBranch(repo1, branchName)
			repo2 := cloneTestRepoOnExistingBranch(ctx, branchName)
			state.Sha = commitAFile(repo2, branchName).String()
			pushBranch(repo2, branchName)

			origPRId, e = provider.CreatePR(ctx, branchName, false, state)
			Expect(e).To(BeNil())
		})

		It("doesn't return an error and ID larger than 0", func() {
			Expect(err).To(BeNil())
			Expect(pr.ID).To(Equal(origPRId))
		})
	})
})

var _ = Describe("GitHubGITProvider GetPRThatCausedCommit", func() {
	ctx := context.Background()
	provider, providerErr := git.NewGitHubGITProvider(ctx, remoteURL, token)

	BeforeEach(func() {
		if remoteURL == "" || token == "" {
			Skip("GITHUB_URL and/or GITHUB_TOKEN environment variables not set")
		}

		if providerErr != nil {
			Fail(fmt.Sprintf("Provider initialization failed: %s", providerErr))
		}
	})

	var err error
	var pr git.PullRequest
	var mergedPR git.PullRequest
	now := time.Now()
	state := &git.PRState{
		Env:   "dev",
		Group: "testgroup",
		App:   "testapp",
		Tag:   now.Format("20060102150405"),
		Sha:   "",
	}

	JustBeforeEach(func() {
		pr, err = provider.GetPRThatCausedCommit(ctx, state.Sha)
	})

	When("Getting PR with empty SHA", func() {
		It("returns an error", func() {
			Expect(err.Error()).To(ContainSubstring("no PR found for sha:"))
		})
	})

	When("Getting PR with existing SHA", func() {
		BeforeEach(func() {
			branchName := randomBranchName("testing-get-pr-that-caused-commit")
			repo1 := cloneTestRepoWithNewBranch(ctx, branchName)
			pushBranch(repo1, branchName)
			repo2 := cloneTestRepoOnExistingBranch(ctx, branchName)
			commitSha := commitAFile(repo2, branchName)
			pushBranch(repo2, branchName)

			_, e := provider.CreatePR(ctx, branchName, false, state)
			Expect(e).To(BeNil())

			mergedPR, e = provider.GetPRWithBranch(ctx, branchName, git.DefaultBranch)
			Expect(e).To(BeNil())

			e = provider.MergePR(ctx, mergedPR.ID, commitSha.String())
			Expect(e).To(BeNil())

			repo3 := cloneTestRepoOnExistingBranch(ctx, git.DefaultBranch)
			mergeSha, e := repo3.GetCurrentCommit()
			Expect(e).To(BeNil())

			state.Sha = mergeSha.String()
		})

		It("doesn't return an error and pr.ID equals mergedPR.ID", func() {
			Expect(err).To(BeNil())
			Expect(pr.ID).To(Equal(mergedPR.ID))
		})
	})
})
