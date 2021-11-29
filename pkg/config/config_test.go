package config

import (
	"bytes"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
		reader := bytes.NewReader([]byte(configData))
		config, err = LoadConfig(reader)
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
})
