package main

import (
	"os"
	"testing"
)

func TestCIRequirements(t *testing.T) {
	ciEnvVar := os.Getenv("CI")
	if ciEnvVar != "true" {
		t.Skipf("CI environment variable not set to true: %s", ciEnvVar)
	}

	reqEnvVars := []string{
		"AZDO_URL",
		"AZDO_PAT",
		"GITHUB_URL",
		"GITHUB_TOKEN",
	}

	for _, envVar := range reqEnvVars {
		v := os.Getenv(envVar)
		if v == "" {
			t.Errorf("%s environment variable is required by CI.", envVar)
		}
	}
}
