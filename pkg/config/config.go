package config

import (
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v2"
)

type FeatureApp struct {
	LabelSelector map[string]string `yaml:"selector"`
}

type Feature map[string]map[string]FeatureApp

func (f Feature) GetFeatureApp(group, app string) (FeatureApp, error) {
	featureGroup, ok := f[group]
	if !ok {
		return FeatureApp{}, fmt.Errorf("feature group not found in configuration: %s", group)
	}
	featureApp, ok := featureGroup[app]
	if !ok {
		return FeatureApp{}, fmt.Errorf("feature app not found in configuration: %s", app)
	}
	return featureApp, nil
}

type Environment struct {
	Name      string `yaml:"name"`
	Automated bool   `yaml:"auto"`
}

type Config struct {
	PRFlow       string        `yaml:"prflow"`
	Environments []Environment `yaml:"environments"`
	Features     Feature       `yaml:"features"`
}

func LoadConfig(file io.Reader) (Config, error) {
	cfg := Config{}
	decoder := yaml.NewDecoder(file)
	err := decoder.Decode(&cfg)
	if len(cfg.Environments) == 0 {
		return Config{}, fmt.Errorf("environments list cannot be empty")
	}
	if cfg.PRFlow == "" {
		cfg.PRFlow = "per-app"
	} else if cfg.PRFlow != "per-app" && cfg.PRFlow != "per-env" {
		return Config{}, fmt.Errorf("invalid prflow value %s", cfg.PRFlow)
	}
	return cfg, err
}

func (c Config) HasNextEnvironment(envName string) bool {
	last := len(c.Environments) - 1
	return c.Environments[last].Name != envName
}

func (c Config) NextEnvironment(envName string) (Environment, error) {
	for i, e := range c.Environments {
		if e.Name == envName {
			return c.Environments[i+1], nil
		}
	}
	return Environment{}, errors.New("could not find next environment")
}

func (c Config) PrevEnvironment(envName string) (Environment, error) {
	for i, e := range c.Environments {
		if e.Name == envName {
			return c.Environments[i-1], nil
		}
	}
	return Environment{}, errors.New("could not find prev environment")
}

func (c Config) IsEnvironmentAutomated(name string) (bool, error) {
	for _, e := range c.Environments {
		if e.Name == name {
			return e.Automated, nil
		}
	}
	return false, fmt.Errorf("could not find environment with name %q", name)
}

func (c Config) IsAnyEnvironmentManual() bool {
	for _, e := range c.Environments {
		if !e.Automated {
			return true
		}
	}
	return false
}
