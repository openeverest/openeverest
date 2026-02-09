// Package readeradapter provides a Casbin adapter for reading policy from an io.Reader source.
package readeradapter

import (
	"errors"
	"io"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"

	rbacutils "github.com/openeverest/openeverest/v2/pkg/rbac/utils"
)

// Adapter reads a policy from an io.Reader source.
type Adapter struct {
	content string
}

func tryUnmarshalFromYAML(content []byte) (bool, string) {
	cm := corev1.ConfigMap{}
	if err := yaml.Unmarshal(content, &cm); err != nil {
		return false, ""
	}
	s, ok := cm.Data["policy.csv"]
	if !ok {
		return false, ""
	}
	return true, s
}

// New returns a new adapter that reads from the given io.Reader.
func New(in io.Reader) (*Adapter, error) {
	content, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}

	var policy string
	if ok, s := tryUnmarshalFromYAML(content); ok {
		policy = s
	} else {
		policy = string(content)
	}

	return &Adapter{
		content: policy,
	}, nil
}

// SetContent sets the content of the adapter.
func (a *Adapter) SetContent(content string) {
	a.content = content
}

// LoadPolicy loads all policy rules from the storage.
func (a *Adapter) LoadPolicy(model model.Model) error {
	strs := strings.Split(a.content, "\n")
	for _, str := range strs {
		if str == "" {
			continue
		}
		if err := rbacutils.LoadPolicyLine(str, model); err != nil {
			return err
		}
	}
	return nil
}

// SavePolicy saves all policy rules to the storage.
func (a *Adapter) SavePolicy(_ model.Model) error {
	return errors.New("not implemented")
}

// AddPolicy adds a policy rule to the storage.
func (a *Adapter) AddPolicy(_ string, _ string, _ []string) error {
	return errors.New("not implemented")
}

// RemovePolicy removes a policy rule from the storage.
func (a *Adapter) RemovePolicy(_ string, _ string, _ []string) error {
	return errors.New("not implemented")
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
func (a *Adapter) RemoveFilteredPolicy(_ string, _ string, _ int, _ ...string) error {
	return errors.New("not implemented")
}
