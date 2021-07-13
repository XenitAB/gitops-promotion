package git

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/google/go-github/v37/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

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
		_, err = NewGitHubGITProvider(ctx, "", "foo")
		Expect(err).To(MatchError("remoteURL empty"))
	})

	It("returns error when creating without token", func() {
		_, err = NewGitHubGITProvider(ctx, "https://github.com/org/repo", "")
		Expect(err).To(MatchError("token empty"))
	})

	It("returns error when creating without github address", func() {
		_, err = NewGitHubGITProvider(ctx, "https://foo.bar/org/repo", "foo")
		Expect(err).To(MatchError("host does not start with https://github.com: https://foo.bar"))
	})

	It("returns error when creating with fake token", func() {
		_, err = NewGitHubGITProvider(ctx, "https://github.com/org/repo", "foo")
		Expect(err).To(MatchError("unable to authenticate using token"))
	})

	It("is successfully created when creating with correct token", func() {
		var provider *GitHubGITProvider
		remoteURL := os.Getenv("GITHUB_URL")
		token := os.Getenv("GITHUB_TOKEN")

		if remoteURL == "" || token == "" {
			Skip("GITHUB_URL and/or GITHUB_TOKEN environment variables not set")
		}

		provider, err = NewGitHubGITProvider(ctx, remoteURL, token)
		Expect(err).To(BeNil())
		Expect(remoteURL).To(ContainSubstring(provider.owner))
		Expect(remoteURL).To(ContainSubstring(provider.repo))
	})
})

var _ = Describe("GitHubGITProvider CreatePR", func() {
	ctx := context.Background()
	remoteURL := os.Getenv("GITHUB_URL")
	token := os.Getenv("GITHUB_TOKEN")
	providerTypeString := string(ProviderTypeGitHub)
	provider, providerErr := NewGitHubGITProvider(ctx, remoteURL, token)

	BeforeEach(func() {
		if remoteURL == "" || token == "" {
			Skip("GITHUB_URL and/or GITHUB_TOKEN environment variables not set")
		}

		if providerErr != nil {
			Fail("Provider initialization failed")
		}
	})

	var err error
	var branchName string
	var auto bool
	var tmpDir string
	now := time.Now()
	state := &PRState{
		Env:   "dev",
		Group: "testgroup",
		App:   "testapp",
		Tag:   now.Format("20060102150405"),
		Sha:   "",
	}

	JustBeforeEach(func() {
		err = provider.CreatePR(ctx, branchName, auto, state)
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
			branchName = "testing-does-not-exist"
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

	When("Creating PR force pushed branchName", func() {
		BeforeEach(func() {
			branchName = "testing-create-pr"

			var e error
			tmpDir, e = ioutil.TempDir("", "testing")
			Expect(e).To(BeNil())

			e = Clone(remoteURL, "pat", token, tmpDir, DefaultBranch)
			Expect(e).To(BeNil())

			repo, e := LoadRepository(ctx, tmpDir, providerTypeString, token)
			Expect(e).To(BeNil())

			e = repo.CreateBranch(branchName, false)
			Expect(e).To(BeNil())

			e = repo.Push(branchName, true)
			Expect(e).To(BeNil())

			e = os.RemoveAll(tmpDir)
			Expect(e).To(BeNil())

			tmpDir, e = ioutil.TempDir("", "testing")
			Expect(e).To(BeNil())

			e = Clone(remoteURL, "pat", token, tmpDir, branchName)
			Expect(e).To(BeNil())

			repo, e = LoadRepository(ctx, tmpDir, providerTypeString, token)
			Expect(e).To(BeNil())

			f, e := os.Create(fmt.Sprintf("%s/%s.txt", tmpDir, branchName))
			Expect(e).To(BeNil())

			_, e = f.WriteString(fmt.Sprintln(time.Now()))
			Expect(e).To(BeNil())

			e = f.Close()
			Expect(e).To(BeNil())

			_, e = repo.CreateCommit(branchName, fmt.Sprintln(time.Now()))
			Expect(e).To(BeNil())

			e = repo.Push(branchName, true)
			Expect(e).To(BeNil())
		})

		AfterEach(func() {
			e := os.RemoveAll(tmpDir)
			Expect(e).To(BeNil())
		})

		It("doesn't return an error", func() {
			Expect(err).To(BeNil())
		})
	})

	When("Creating PR with existing PR and branchName", func() {
		BeforeEach(func() {
			branchName = "testing-create-pr"

			var e error
			tmpDir, e = ioutil.TempDir("", "testing")
			Expect(e).To(BeNil())

			e = Clone(remoteURL, "pat", token, tmpDir, branchName)
			Expect(e).To(BeNil())

			repo, e := LoadRepository(ctx, tmpDir, providerTypeString, token)
			Expect(e).To(BeNil())

			_, e = repo.GetPRForCurrentBranch(ctx)
			Expect(e).To(BeNil())
		})

		AfterEach(func() {
			e := os.RemoveAll(tmpDir)
			Expect(e).To(BeNil())
		})

		It("doesn't return an error", func() {
			Expect(err).To(BeNil())
		})
	})
})

var _ = Describe("GitHubGITProvider GetStatus", func() {
	ctx := context.Background()
	remoteURL := os.Getenv("GITHUB_URL")
	token := os.Getenv("GITHUB_TOKEN")
	providerTypeString := string(ProviderTypeGitHub)
	provider, providerErr := NewGitHubGITProvider(ctx, remoteURL, token)

	BeforeEach(func() {
		if remoteURL == "" || token == "" {
			Skip("GITHUB_URL and/or GITHUB_TOKEN environment variables not set")
		}

		if providerErr != nil {
			Fail("Provider initialization failed")
		}
	})

	var err error
	var status Status
	var tmpDir string
	now := time.Now()
	state := &PRState{
		Env:   "dev",
		Group: "testgroup",
		App:   "testapp",
		Tag:   now.Format("20060102150405"),
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
		BeforeEach(func() {
			var e error
			tmpDir, e = ioutil.TempDir("", "testing")
			Expect(e).To(BeNil())

			e = Clone(remoteURL, "pat", token, tmpDir, DefaultBranch)
			Expect(e).To(BeNil())

			repo, e := LoadRepository(ctx, tmpDir, providerTypeString, token)
			Expect(e).To(BeNil())

			f, e := os.Create(fmt.Sprintf("%s/%s.txt", tmpDir, DefaultBranch))
			Expect(e).To(BeNil())

			_, e = f.WriteString(fmt.Sprintln(time.Now()))
			Expect(e).To(BeNil())

			e = f.Close()
			Expect(e).To(BeNil())

			_, e = repo.CreateCommit(DefaultBranch, fmt.Sprintln(time.Now()))
			Expect(e).To(BeNil())

			e = repo.Push(DefaultBranch, true)
			Expect(e).To(BeNil())

			sha, e := repo.GetCurrentCommit()
			Expect(e).To(BeNil())

			state.Sha = sha.String()
		})

		AfterEach(func() {
			e := os.RemoveAll(tmpDir)
			Expect(e).To(BeNil())
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
	remoteURL := os.Getenv("GITHUB_URL")
	token := os.Getenv("GITHUB_TOKEN")
	providerTypeString := string(ProviderTypeGitHub)
	provider, providerErr := NewGitHubGITProvider(ctx, remoteURL, token)

	BeforeEach(func() {
		if remoteURL == "" || token == "" {
			Skip("GITHUB_URL and/or GITHUB_TOKEN environment variables not set")
		}

		if providerErr != nil {
			Fail("Provider initialization failed")
		}
	})

	var err error
	var tmpDir string
	var branchName string
	var prID int
	now := time.Now()
	state := &PRState{
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
			branchName = "testing-merge-pr"

			var e error
			tmpDir, e = ioutil.TempDir("", "testing")
			Expect(e).To(BeNil())

			e = Clone(remoteURL, "pat", token, tmpDir, DefaultBranch)
			Expect(e).To(BeNil())

			repo, e := LoadRepository(ctx, tmpDir, providerTypeString, token)
			Expect(e).To(BeNil())

			e = repo.CreateBranch(branchName, true)
			Expect(e).To(BeNil())

			e = repo.Push(branchName, true)
			Expect(e).To(BeNil())

			e = os.RemoveAll(tmpDir)
			Expect(e).To(BeNil())

			tmpDir, e = ioutil.TempDir("", "testing")
			Expect(e).To(BeNil())

			e = Clone(remoteURL, "pat", token, tmpDir, branchName)
			Expect(e).To(BeNil())

			repo, e = LoadRepository(ctx, tmpDir, providerTypeString, token)
			Expect(e).To(BeNil())

			f, e := os.Create(fmt.Sprintf("%s/%s.txt", tmpDir, branchName))
			Expect(e).To(BeNil())

			_, e = f.WriteString(fmt.Sprintln(time.Now()))
			Expect(e).To(BeNil())

			e = f.Close()
			Expect(e).To(BeNil())

			sha, e := repo.CreateCommit(branchName, fmt.Sprintln(time.Now()))
			Expect(e).To(BeNil())

			state.Sha = sha.String()

			e = repo.Push(branchName, true)
			Expect(e).To(BeNil())

			e = provider.CreatePR(ctx, branchName, false, state)
			Expect(e).To(BeNil())

			pr, e := provider.GetPRWithBranch(ctx, branchName, DefaultBranch)
			Expect(e).To(BeNil())

			prID = pr.ID
		})

		AfterEach(func() {
			e := os.RemoveAll(tmpDir)
			Expect(e).To(BeNil())
		})

		It("doesn't return an error", func() {
			Expect(err).To(BeNil())
		})
	})
})

var _ = Describe("GitHubGITProvider GetPRWithBranch", func() {
	ctx := context.Background()
	remoteURL := os.Getenv("GITHUB_URL")
	token := os.Getenv("GITHUB_TOKEN")
	providerTypeString := string(ProviderTypeGitHub)
	provider, providerErr := NewGitHubGITProvider(ctx, remoteURL, token)

	BeforeEach(func() {
		if remoteURL == "" || token == "" {
			Skip("GITHUB_URL and/or GITHUB_TOKEN environment variables not set")
		}

		if providerErr != nil {
			Fail("Provider initialization failed")
		}
	})

	var err error
	var tmpDir string
	var branchName string
	var pr PullRequest
	now := time.Now()
	state := &PRState{
		Env:   "dev",
		Group: "testgroup",
		App:   "testapp",
		Tag:   now.Format("20060102150405"),
		Sha:   "",
	}

	JustBeforeEach(func() {
		pr, err = provider.GetPRWithBranch(ctx, branchName, DefaultBranch)
	})

	When("Getting PR with empty branchName", func() {
		It("returns an error", func() {
			Expect(err.Error()).To(ContainSubstring("no PR found for branches"))
		})
	})

	When("Getting PR with existing branchName", func() {
		BeforeEach(func() {
			branchName = "testing-get-pr-with-branch"

			var e error
			tmpDir, e = ioutil.TempDir("", "testing")
			Expect(e).To(BeNil())

			e = Clone(remoteURL, "pat", token, tmpDir, DefaultBranch)
			Expect(e).To(BeNil())

			repo, e := LoadRepository(ctx, tmpDir, providerTypeString, token)
			Expect(e).To(BeNil())

			e = repo.CreateBranch(branchName, true)
			Expect(e).To(BeNil())

			e = repo.Push(branchName, true)
			Expect(e).To(BeNil())

			e = os.RemoveAll(tmpDir)
			Expect(e).To(BeNil())

			tmpDir, e = ioutil.TempDir("", "testing")
			Expect(e).To(BeNil())

			e = Clone(remoteURL, "pat", token, tmpDir, branchName)
			Expect(e).To(BeNil())

			repo, e = LoadRepository(ctx, tmpDir, providerTypeString, token)
			Expect(e).To(BeNil())

			f, e := os.Create(fmt.Sprintf("%s/%s.txt", tmpDir, branchName))
			Expect(e).To(BeNil())

			_, e = f.WriteString(fmt.Sprintln(time.Now()))
			Expect(e).To(BeNil())

			e = f.Close()
			Expect(e).To(BeNil())

			sha, e := repo.CreateCommit(branchName, fmt.Sprintln(time.Now()))
			Expect(e).To(BeNil())

			state.Sha = sha.String()

			e = repo.Push(branchName, true)
			Expect(e).To(BeNil())

			e = provider.CreatePR(ctx, branchName, false, state)
			Expect(e).To(BeNil())
		})

		AfterEach(func() {
			e := os.RemoveAll(tmpDir)
			Expect(e).To(BeNil())
		})

		It("doesn't return an error and ID larger than 0", func() {
			Expect(err).To(BeNil())
			Expect(pr.ID).To(BeNumerically(">", 0))
		})
	})
})

var _ = Describe("GitHubGITProvider GetPRThatCausedCommit", func() {
	ctx := context.Background()
	remoteURL := os.Getenv("GITHUB_URL")
	token := os.Getenv("GITHUB_TOKEN")
	providerTypeString := string(ProviderTypeGitHub)
	provider, providerErr := NewGitHubGITProvider(ctx, remoteURL, token)

	BeforeEach(func() {
		if remoteURL == "" || token == "" {
			Skip("GITHUB_URL and/or GITHUB_TOKEN environment variables not set")
		}

		if providerErr != nil {
			Fail("Provider initialization failed")
		}
	})

	var err error
	var tmpDir string
	var branchName string
	var pr PullRequest
	var mergedPR PullRequest
	now := time.Now()
	state := &PRState{
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
			branchName = "testing-get-pr-that-caused-commit"

			var e error
			tmpDir, e = ioutil.TempDir("", "testing")
			Expect(e).To(BeNil())

			e = Clone(remoteURL, "pat", token, tmpDir, DefaultBranch)
			Expect(e).To(BeNil())

			repo, e := LoadRepository(ctx, tmpDir, providerTypeString, token)
			Expect(e).To(BeNil())

			e = repo.CreateBranch(branchName, true)
			Expect(e).To(BeNil())

			e = repo.Push(branchName, true)
			Expect(e).To(BeNil())

			e = os.RemoveAll(tmpDir)
			Expect(e).To(BeNil())

			tmpDir, e = ioutil.TempDir("", "testing")
			Expect(e).To(BeNil())

			e = Clone(remoteURL, "pat", token, tmpDir, branchName)
			Expect(e).To(BeNil())

			repo, e = LoadRepository(ctx, tmpDir, providerTypeString, token)
			Expect(e).To(BeNil())

			f, e := os.Create(fmt.Sprintf("%s/%s.txt", tmpDir, branchName))
			Expect(e).To(BeNil())

			_, e = f.WriteString(fmt.Sprintln(time.Now()))
			Expect(e).To(BeNil())

			e = f.Close()
			Expect(e).To(BeNil())

			commitSha, e := repo.CreateCommit(branchName, fmt.Sprintln(time.Now()))
			Expect(e).To(BeNil())

			e = repo.Push(branchName, true)
			Expect(e).To(BeNil())

			e = provider.CreatePR(ctx, branchName, false, state)
			Expect(e).To(BeNil())

			mergedPR, e = provider.GetPRWithBranch(ctx, branchName, DefaultBranch)
			Expect(e).To(BeNil())

			e = provider.MergePR(ctx, mergedPR.ID, commitSha.String())
			Expect(e).To(BeNil())

			tmpDir, e = ioutil.TempDir("", "testing")
			Expect(e).To(BeNil())

			e = Clone(remoteURL, "pat", token, tmpDir, DefaultBranch)
			Expect(e).To(BeNil())

			repo, e = LoadRepository(ctx, tmpDir, providerTypeString, token)
			Expect(e).To(BeNil())

			mergeSha, e := repo.GetCurrentCommit()
			Expect(e).To(BeNil())

			state.Sha = mergeSha.String()
		})

		AfterEach(func() {
			e := os.RemoveAll(tmpDir)
			Expect(e).To(BeNil())
		})

		It("doesn't return an error and pr.ID equals mergedPR.ID", func() {
			Expect(err).To(BeNil())
			Expect(pr.ID).To(Equal(mergedPR.ID))
		})
	})
})
