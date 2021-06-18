package git

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/google/go-github/v35/github"
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
	now := time.Now()
	state := &PRState{
		Env:   "dev",
		Group: "testgroup",
		App:   "testapp",
		Tag:   now.Format("20060102150405"),
		Sha:   "testsha",
	}

	JustBeforeEach(func() {
		err = provider.CreatePR(ctx, branchName, auto, state)
	})

	When("Creating PR with empty values", func() {
		It("returns error", func() {
			gitHubError, ok := err.(*github.ErrorResponse)
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
			branchName = "test"
		})

		It("returns error", func() {
			gitHubError, ok := err.(*github.ErrorResponse)
			Expect(ok).To(Equal(true))
			body, bodyErr := ioutil.ReadAll(gitHubError.Response.Body)
			Expect(bodyErr).To(BeNil())
			bodyErr = gitHubError.Response.Body.Close()
			Expect(bodyErr).To(BeNil())

			Expect(string(body)).To(ContainSubstring("{\"resource\":\"PullRequest\",\"code\":\"missing_field\",\"field\":\"head\"}"))
		})
	})
})
