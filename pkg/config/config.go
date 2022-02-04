package config

import (
	"fmt"
	"io"
	"time"

	"gopkg.in/yaml.v2"
)

type PRFlowType string

const (
	PRFlowTypePerApp PRFlowType = "per-app"
	PRFlowTypePerEnv PRFlowType = "per-env"
)

type App struct {
	FeatureLabelSelector map[string]string `yaml:"featureLabelSelector"`
}

type Group struct {
	Applications map[string]App `yaml:"applications"`
}

type Environment struct {
	Name      string `yaml:"name"`
	Automated bool   `yaml:"auto"`
}

type Config struct {
	PRFlow        PRFlowType       `yaml:"prflow"`
	StatusTimeout time.Duration    `yaml:"status_timeout_minutes"`
	Environments  []Environment    `yaml:"environments"`
	Groups        map[string]Group `yaml:"groups"`
}

func LoadConfig(file io.Reader) (Config, error) {
	cfg := Config{}
	decoder := yaml.NewDecoder(file)
	err := decoder.Decode(&cfg)
	if err != nil {
		return Config{}, err
	}

	if len(cfg.Environments) == 0 {
		return Config{}, fmt.Errorf("environments list cannot be empty")
	}
	if cfg.PRFlow == "" {
		cfg.PRFlow = PRFlowTypePerApp
	}
	if cfg.StatusTimeout.String() == (0 * time.Minute).String() {
		cfg.StatusTimeout = 5 * time.Minute
	}
	switch cfg.PRFlow {
	case PRFlowTypePerApp, PRFlowTypePerEnv:
		break
	default:
		return Config{}, fmt.Errorf("invalid prflow value: %s", cfg.PRFlow)
	}

	return cfg, nil
}

func (c Config) HasNextEnvironment(name string) bool {
	last := len(c.Environments) - 1
	return c.Environments[last].Name != name
}

func (c Config) NextEnvironment(name string) (Environment, error) {
	_, i, err := c.getEnvironment(name)
	if err != nil {
		return Environment{}, err
	}
	if i == len(c.Environments)-1 {
		return Environment{}, fmt.Errorf("last environment cannot have a next environment")
	}
	return c.Environments[i+1], nil
}

func (c Config) PrevEnvironment(name string) (Environment, error) {
	_, i, err := c.getEnvironment(name)
	if err != nil {
		return Environment{}, err
	}
	if i == 0 {
		return Environment{}, fmt.Errorf("first environment cannot have a previous environment")
	}
	return c.Environments[i-1], nil
}

func (c Config) IsEnvironmentAutomated(name string) (bool, error) {
	e, _, err := c.getEnvironment(name)
	if err != nil {
		return false, err
	}
	return e.Automated, nil
}

func (c Config) IsAnyEnvironmentManual() bool {
	for _, e := range c.Environments {
		if !e.Automated {
			return true
		}
	}
	return false
}

func (c Config) GetFeatureLabelSelector(group, app string) (map[string]string, error) {
	groupObj, ok := c.Groups[group]
	if !ok {
		return nil, fmt.Errorf("configuration does not contain group %s", group)
	}
	appObj, ok := groupObj.Applications[app]
	if !ok {
		return nil, fmt.Errorf("configuration group %s does not contain app %s", group, app)
	}
	return appObj.FeatureLabelSelector, nil
}

func (c Config) getEnvironment(name string) (Environment, int, error) {
	for i, e := range c.Environments {
		if e.Name == name {
			return e, i, nil
		}
	}
	return Environment{}, 0, fmt.Errorf("environment named %s does not exist", name)
}
