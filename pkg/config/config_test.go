package config

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigSimple(t *testing.T) {
	data := `
    environments:
      - name: dev
        auto: true
      - name: qa
        auto: true
      - name: prod
        auto: false
  `
	reader := bytes.NewReader([]byte(data))
	cfg, err := LoadConfig(reader)
	require.NoError(t, err)

	require.Len(t, cfg.Environments, 3)
	require.Equal(t, PRFlowTypePerApp, cfg.PRFlow)
	require.True(t, cfg.IsAnyEnvironmentManual())

	require.True(t, cfg.HasNextEnvironment("dev"))
	require.True(t, cfg.HasNextEnvironment("qa"))
	require.False(t, cfg.HasNextEnvironment("prod"))

	_, err = cfg.IsEnvironmentAutomated("foobar")
	require.EqualError(t, err, "environment named foobar does not exist")
	automated, err := cfg.IsEnvironmentAutomated("dev")
	require.NoError(t, err)
	require.True(t, automated)
	automated, err = cfg.IsEnvironmentAutomated("qa")
	require.NoError(t, err)
	require.True(t, automated)
	automated, err = cfg.IsEnvironmentAutomated("prod")
	require.NoError(t, err)
	require.False(t, automated)

	_, err = cfg.NextEnvironment("foobar")
	require.EqualError(t, err, "environment named foobar does not exist")
	next, err := cfg.NextEnvironment("dev")
	require.NoError(t, err)
	require.Equal(t, "qa", next.Name)
	next, err = cfg.NextEnvironment("qa")
	require.NoError(t, err)
	require.Equal(t, "prod", next.Name)
	next, err = cfg.NextEnvironment("prod")
	require.EqualError(t, err, "last environment cannot have a next environment")

	_, err = cfg.PrevEnvironment("foobar")
	require.EqualError(t, err, "environment named foobar does not exist")
	prev, err := cfg.PrevEnvironment("prod")
	require.NoError(t, err)
	require.Equal(t, "qa", prev.Name)
	prev, err = cfg.PrevEnvironment("qa")
	require.NoError(t, err)
	require.Equal(t, "dev", prev.Name)
	prev, err = cfg.PrevEnvironment("dev")
	require.EqualError(t, err, "first environment cannot have a previous environment")
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
