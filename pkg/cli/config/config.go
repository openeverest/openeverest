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

// Config is the top-level credential configuration file.
type Config struct {
	APIVersion     string         `yaml:"apiVersion"`
	Kind           string         `yaml:"kind"`
	CurrentContext string         `yaml:"currentContext,omitempty"`
	Contexts       []NamedContext `yaml:"contexts,omitempty"`
	Servers        []NamedServer  `yaml:"servers,omitempty"`
	Users          []NamedUser    `yaml:"users,omitempty"`
}

// NamedContext pairs a name with a Context.
type NamedContext struct {
	Name    string  `yaml:"name"`
	Context Context `yaml:"context"`
}

// Context links a named server to a named user.
type Context struct {
	Server string `yaml:"server"`
	User   string `yaml:"user"`
}

// NamedServer pairs a name with a Server.
type NamedServer struct {
	Name   string `yaml:"name"`
	Server Server `yaml:"server"`
}

// Server holds the URL of an Everest API endpoint.
type Server struct {
	URL string `yaml:"url"`
}

// NamedUser pairs a name with a User.
type NamedUser struct {
	Name string `yaml:"name"`
	User User   `yaml:"user"`
}

// User holds the credentials for an Everest account.
type User struct {
	AccessToken  string    `yaml:"accessToken"`
	RefreshToken string    `yaml:"refreshToken"`
	ExpiresAt    time.Time `yaml:"expiresAt"`
}

// DefaultPath returns the default path to the config file.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not determine config directory: %w", err)
	}
	return filepath.Join(dir, "everest", "config.yaml"), nil
}

// Load reads the config file at path. Returns an empty Config if the file does not exist.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{APIVersion: "config.openeverest.io/v1alpha1", Kind: "ClientConfig"}, nil
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

// UpsertContext inserts or overwrites the named context.
func (c *Config) UpsertContext(name string, ctx Context) {
	for i, nc := range c.Contexts {
		if nc.Name == name {
			c.Contexts[i].Context = ctx
			return
		}
	}
	c.Contexts = append(c.Contexts, NamedContext{Name: name, Context: ctx})
}

// UpsertServer inserts or overwrites the named server.
func (c *Config) UpsertServer(name string, s Server) {
	for i, ns := range c.Servers {
		if ns.Name == name {
			c.Servers[i].Server = s
			return
		}
	}
	c.Servers = append(c.Servers, NamedServer{Name: name, Server: s})
}

// UpsertUser inserts or overwrites the named user.
func (c *Config) UpsertUser(name string, u User) {
	for i, nu := range c.Users {
		if nu.Name == name {
			c.Users[i].User = u
			return
		}
	}
	c.Users = append(c.Users, NamedUser{Name: name, User: u})
}

// GetCurrentContext returns the context referenced by CurrentContext.
func (c *Config) GetCurrentContext() (Context, bool) {
	for _, nc := range c.Contexts {
		if nc.Name == c.CurrentContext {
			return nc.Context, true
		}
	}
	return Context{}, false
}

// GetServer returns the server with the given name.
func (c *Config) GetServer(name string) (Server, bool) {
	for _, ns := range c.Servers {
		if ns.Name == name {
			return ns.Server, true
		}
	}
	return Server{}, false
}

// GetUser returns the user with the given name.
func (c *Config) GetUser(name string) (User, bool) {
	for _, nu := range c.Users {
		if nu.Name == name {
			return nu.User, true
		}
	}
	return User{}, false
}

// RemoveContext removes the named context from the config.
func (c *Config) RemoveContext(name string) {
	out := c.Contexts[:0]
	for _, nc := range c.Contexts {
		if nc.Name != name {
			out = append(out, nc)
		}
	}
	c.Contexts = out
}

// RemoveServer removes the named server from the config.
func (c *Config) RemoveServer(name string) {
	out := c.Servers[:0]
	for _, ns := range c.Servers {
		if ns.Name != name {
			out = append(out, ns)
		}
	}
	c.Servers = out
}

// RemoveUser removes the named user from the config.
func (c *Config) RemoveUser(name string) {
	out := c.Users[:0]
	for _, nu := range c.Users {
		if nu.Name != name {
			out = append(out, nu)
		}
	}
	c.Users = out
}

// IsServerReferenced reports whether any context references the named server.
func (c *Config) IsServerReferenced(name string) bool {
	for _, nc := range c.Contexts {
		if nc.Context.Server == name {
			return true
		}
	}
	return false
}

// IsUserReferenced reports whether any context references the named user.
func (c *Config) IsUserReferenced(name string) bool {
	for _, nc := range c.Contexts {
		if nc.Context.User == name {
			return true
		}
	}
	return false
}
