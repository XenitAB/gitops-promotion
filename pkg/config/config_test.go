package config

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

var environmentList = "environments:\n  - name: dev\n    auto: true\n"

func TestNewGitHubGITProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config")
}

var _ = Describe("Config", func() {
	var err error
	var config Config
	var configData string = "{}"

	JustBeforeEach(func() {
		path := "gitops-promotion.yaml"
		fs := afero.NewMemMapFs()
		afero.WriteFile(fs, path, []byte(configData), 0600)
		config, err = LoadConfig(fs, path)
	})

	It("returns an error when mandatory values are missing", func() {
		Expect(err).NotTo(BeNil())
	})

	Describe("With valid environments list", func() {
		BeforeEach(func() {
			configData = environmentList
		})

		It("presents environments", func() {
			Expect(config.Environments).To(ContainElement(Environment{Name: "dev", Automated: true}))
		})
	})

	Describe("Looking at prflow", func() {
		It("defaults to per-app", func() {
			Expect(config.PRFlow).To(Equal("per-app"))
		})

		Describe("When given per-env", func() {
			BeforeEach(func() {
				configData = environmentList + "prflow: per-env\n"
			})

			It("yields per-env", func() {
				Expect(config.PRFlow).To(Equal("per-env"))
			})
		})

		Describe("When given invalid input", func() {
			BeforeEach(func() {
				configData = environmentList + "prflow: nonsense\n"
			})

			It("throws an error", func() {
				Expect(err).NotTo(BeNil())
			})
		})
	})
})
