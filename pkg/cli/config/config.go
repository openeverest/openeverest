// everest
// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package config manages the everestctl credential configuration file.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	configDir  = ".config/everest"
	configFile = "config.yaml"
)

// Config is the top-level credential configuration file.
type Config struct {
	APIVersion     string         `yaml:"apiVersion"`
	Kind           string         `yaml:"kind"`
	CurrentContext string         `yaml:"currentContext"`
	Contexts       []NamedContext `yaml:"contexts,omitempty"`
}

// NamedContext pairs a name with a Context.
type NamedContext struct {
	Name    string  `yaml:"name"`
	Context Context `yaml:"context"`
}

// Context holds the server URL and credentials for one Everest server.
type Context struct {
	Server       string    `yaml:"server"`
	AccessToken  string    `yaml:"accessToken"`
	RefreshToken string    `yaml:"refreshToken"`
	ExpiresAt    time.Time `yaml:"expiresAt"`
}

// DefaultPath returns the default path to the config file (~/.config/everest/config.yaml).
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

// Load reads the config file at path. Returns an empty Config if the file does not exist.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{APIVersion: "v1", Kind: "Config"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to path with mode 0600, creating parent directories as needed.
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

// UpsertContext inserts or overwrites the named context in the config.
func (c *Config) UpsertContext(name string, ctx Context) {
	for i, nc := range c.Contexts {
		if nc.Name == name {
			c.Contexts[i].Context = ctx
			return
		}
	}
	c.Contexts = append(c.Contexts, NamedContext{Name: name, Context: ctx})
}
