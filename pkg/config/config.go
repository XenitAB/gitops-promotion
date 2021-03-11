package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	GitConfig    GitConfig     `yaml:"git"`
	Environments []Environment `yaml:"environments"`
}

type GitConfig struct {
	DefaultBranch string `yaml:"defaultBranch"`
	RemoteName    string `yaml:"remoteName"`
	User          string `yaml:"user"`
}

type Environment struct {
	Name      string `yaml:"name"`
	Automated bool   `yaml:"auto"`
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(fmt.Sprintf("%s/gitops-promotion.yaml", path))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{}
	err = yaml.Unmarshal([]byte(data), &cfg)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) PrevEnvironment(envName string) (Environment, error) {
	for i, e := range c.Environments {
		if e.Name == envName {
			return c.Environments[i-1], nil
		}
	}
	return Environment{}, errors.New("could not find prev environment")
}

func (c Config) NextEnvironment(envName string) (Environment, error) {
	for i, e := range c.Environments {
		if e.Name == envName {
			return c.Environments[i+1], nil
		}
	}
	return Environment{}, errors.New("could not find next environment")
}

func (c Config) HasNextEnvironment(envName string) bool {
	last := len(c.Environments) - 1
	return c.Environments[last].Name != envName
}

func (c Config) IsEnvironmentAutomated(name string) (bool, error) {
	for _, e := range c.Environments {
		if e.Name == name {
			return e.Automated, nil
		}
	}

	return false, fmt.Errorf("could not find environment with name %q", name)
}
