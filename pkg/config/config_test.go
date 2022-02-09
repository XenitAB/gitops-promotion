package config

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const simpleData = `
environments:
  - name: dev
    auto: true
  - name: qa
    auto: true
  - name: prod
    auto: false
`

func TestConfigParse(t *testing.T) {
	reader := bytes.NewReader([]byte(simpleData))
	cfg, err := LoadConfig(reader)
	require.NoError(t, err)
	require.Len(t, cfg.Environments, 3)
	require.Equal(t, PRFlowTypePerApp, cfg.PRFlow)
	require.True(t, cfg.IsAnyEnvironmentManual())
}

func TestConfigHasNext(t *testing.T) {
	reader := bytes.NewReader([]byte(simpleData))
	cfg, err := LoadConfig(reader)
	require.NoError(t, err)
	cases := []struct {
		environment string
		hasNext     bool
	}{
		{
			environment: "dev",
			hasNext:     true,
		},
		{
			environment: "qa",
			hasNext:     true,
		},
		{
			environment: "prod",
			hasNext:     false,
		},
	}
	for _, c := range cases {
		t.Run(c.environment, func(t *testing.T) {
			require.Equal(t, c.hasNext, cfg.HasNextEnvironment(c.environment))
		})
	}
}

func TestConfigIsAutomated(t *testing.T) {
	reader := bytes.NewReader([]byte(simpleData))
	cfg, err := LoadConfig(reader)
	require.NoError(t, err)
	cases := []struct {
		environment string
		isAutomated bool
	}{
		{
			environment: "dev",
			isAutomated: true,
		},
		{
			environment: "qa",
			isAutomated: true,
		},
		{
			environment: "prod",
			isAutomated: false,
		},
	}
	for _, c := range cases {
		t.Run(c.environment, func(t *testing.T) {
			automated, err := cfg.IsEnvironmentAutomated(c.environment)
			require.NoError(t, err)
			require.Equal(t, c.isAutomated, automated)
		})
	}
}

func TestConfigNexPrev(t *testing.T) {
	reader := bytes.NewReader([]byte(simpleData))
	cfg, err := LoadConfig(reader)
	require.NoError(t, err)
	cases := []struct {
		environment     string
		nextEnvironment string
		prevEnvironment string
	}{
		{
			environment:     "dev",
			nextEnvironment: "qa",
		},
		{
			environment:     "qa",
			nextEnvironment: "prod",
			prevEnvironment: "dev",
		},
		{
			environment:     "prod",
			prevEnvironment: "qa",
		},
	}
	for _, c := range cases {
		t.Run(c.environment, func(t *testing.T) {
			e, err := cfg.NextEnvironment(c.environment)
			if cfg.Environments[len(cfg.Environments)-1].Name == c.environment {
				require.EqualError(t, err, "last environment cannot have a next environment")
			} else {
				require.Equal(t, c.nextEnvironment, e.Name)
			}

			e, err = cfg.PrevEnvironment(c.environment)
			if cfg.Environments[0].Name == c.environment {
				require.EqualError(t, err, "first environment cannot have a previous environment")
			} else {
				require.Equal(t, c.prevEnvironment, e.Name)
			}
		})
	}
}

func TestConfigNotFound(t *testing.T) {
	reader := bytes.NewReader([]byte(simpleData))
	cfg, err := LoadConfig(reader)
	require.NoError(t, err)
	_, err = cfg.IsEnvironmentAutomated("foobar")
	require.EqualError(t, err, "environment named foobar does not exist")
	_, err = cfg.NextEnvironment("foobar")
	require.EqualError(t, err, "environment named foobar does not exist")
	_, err = cfg.PrevEnvironment("foobar")
	require.EqualError(t, err, "environment named foobar does not exist")
}

func TestConfigEmptyEnvironments(t *testing.T) {
	data := `
    environments: []
  `
	reader := bytes.NewReader([]byte(data))
	_, err := LoadConfig(reader)
	require.EqualError(t, err, "environments list cannot be empty")
}

func TestConfigPRFlowEnv(t *testing.T) {
	data := `
    environments:
      - name: dev
        auto: true
    prflow: per-env
  `
	reader := bytes.NewReader([]byte(data))
	cfg, err := LoadConfig(reader)
	require.NoError(t, err)
	require.Equal(t, PRFlowTypePerEnv, cfg.PRFlow)
}

func TestConfigPRFlowInvalid(t *testing.T) {
	data := `
    environments:
      - name: dev
        auto: true
    prflow: foobar
  `
	reader := bytes.NewReader([]byte(data))
	_, err := LoadConfig(reader)
	require.EqualError(t, err, "invalid prflow value: foobar")
}

func TestConfigStatusTimeout(t *testing.T) {
	data := `
    environments:
      - name: dev
        auto: true
  `
	reader := bytes.NewReader([]byte(data))
	cfg, err := LoadConfig(reader)
	require.NoError(t, err)
	require.Equal(t, 5*time.Minute, cfg.StatusTimeout)
}

func TestConfigFeature(t *testing.T) {
	data := `
    environments:
      - name: dev
        auto: true
    groups:
      apps:
        applications:
          podinfo:
            featureLabelSelector:
              app: podinfo

  `
	reader := bytes.NewReader([]byte(data))
	cfg, err := LoadConfig(reader)
	require.NoError(t, err)
	require.NotEmpty(t, cfg.Groups)

	featureLabelSelector, err := cfg.GetFeatureLabelSelector("apps", "podinfo")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"app": "podinfo"}, featureLabelSelector)
}
