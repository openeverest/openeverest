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

package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	corev1 "k8s.io/api/core/v1"
	sigsjson "sigs.k8s.io/json"
)

// SecretPatch is a merge patch (RFC 7396) of a Secret's data as the HTTP API
// accepts it. Its document may hold any member of a Secret, so a Secret
// returned by GET is a valid patch, but only the data changes.
type SecretPatch struct {
	// ResourceVersion is the version of the Secret the patch was written
	// against. It is required.
	ResourceVersion string
	// Set holds the keys the patch sets, from data and stringData.
	Set map[string][]byte
	// Remove holds the keys the patch removes with data: {key: null}.
	Remove []string
	// Labels, Type and Immutable are what the patch sends for those members,
	// nil when it leaves them out. A nil label value removes the label.
	Labels    map[string]*string
	Type      *corev1.SecretType
	Immutable *bool
}

// ParseSecretPatch decodes a merge patch of a Secret. It refuses a member a
// Secret does not have, so a misspelt key fails instead of changing nothing,
// and a patch without metadata.resourceVersion. Errors name keys, never values.
func ParseSecretPatch(patch []byte) (*SecretPatch, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(patch, &doc); err != nil || doc == nil {
		return nil, errors.New("patch must be a JSON object")
	}

	// Strict and case-sensitive, as Kubernetes decodes: encoding/json would
	// take "stringdata" for stringData here and then find nothing under it.
	strictErrs, err := sigsjson.UnmarshalStrict(patch, &corev1.Secret{},
		sigsjson.DisallowUnknownFields, sigsjson.DisallowDuplicateFields,
	)
	if err == nil {
		err = errors.Join(strictErrs...)
	}
	if err != nil {
		return nil, fmt.Errorf("patch is not a Secret: %w", err)
	}

	p := &SecretPatch{Set: map[string][]byte{}}
	for _, parse := range []func(map[string]json.RawMessage) error{
		p.parseMetadata, p.parseData, p.parseStringData, p.parseTypeAndImmutable,
	} {
		if err := parse(doc); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// Apply returns data with the patch's keys set and removed; data is not
// modified.
func (p *SecretPatch) Apply(data map[string][]byte) map[string][]byte {
	result := maps.Clone(data)
	if result == nil {
		result = map[string][]byte{}
	}
	for _, key := range p.Remove {
		delete(result, key)
	}
	maps.Copy(result, p.Set)
	return result
}

// DataMergePatch returns the merge patch that applies p to a Secret in
// Kubernetes: the resourceVersion as a precondition and the data keys to set
// or remove, nothing else.
func (p *SecretPatch) DataMergePatch() ([]byte, error) {
	data := make(map[string]any, len(p.Set)+len(p.Remove))
	for _, key := range p.Remove {
		data[key] = nil
	}
	for key, value := range p.Set {
		data[key] = value
	}
	return json.Marshal(map[string]any{
		"metadata": map[string]any{"resourceVersion": p.ResourceVersion},
		"data":     data,
	})
}

func (p *SecretPatch) parseMetadata(doc map[string]json.RawMessage) error {
	var metadata struct {
		ResourceVersion string          `json:"resourceVersion"`
		Labels          json.RawMessage `json:"labels"`
	}
	if raw := doc["metadata"]; len(raw) > 0 && !isNull(raw) {
		if err := json.Unmarshal(raw, &metadata); err != nil {
			return fmt.Errorf("metadata: %w", err)
		}
	}
	if metadata.ResourceVersion == "" {
		return errors.New("metadata.resourceVersion is required: send the resourceVersion of the Secret you read, so that a change made since is not overwritten")
	}
	p.ResourceVersion = metadata.ResourceVersion

	if metadata.Labels != nil {
		if err := decodeObject(metadata.Labels, "metadata.labels", &p.Labels); err != nil {
			return err
		}
	}
	return nil
}

func (p *SecretPatch) parseData(doc map[string]json.RawMessage) error {
	raw, found := doc["data"]
	if !found {
		return nil
	}
	var data map[string]*[]byte
	if err := decodeObject(raw, "data", &data); err != nil {
		return err
	}
	for _, key := range slices.Sorted(maps.Keys(data)) {
		if data[key] == nil {
			p.Remove = append(p.Remove, key)
			continue
		}
		p.Set[key] = *data[key]
	}
	return nil
}

// parseStringData runs after parseData: stringData is merged into data on
// write and wins over it, as in Kubernetes.
func (p *SecretPatch) parseStringData(doc map[string]json.RawMessage) error {
	raw, found := doc["stringData"]
	if !found {
		return nil
	}
	var stringData map[string]*string
	if err := decodeObject(raw, "stringData", &stringData); err != nil {
		return err
	}
	for key, value := range stringData {
		if value == nil {
			return fmt.Errorf("stringData[%q] is null; remove a key with data: {%q: null}", key, key)
		}
		p.Set[key] = []byte(*value)
	}
	p.Remove = slices.DeleteFunc(p.Remove, func(key string) bool {
		_, set := p.Set[key]
		return set
	})
	if len(p.Remove) == 0 {
		p.Remove = nil
	}
	return nil
}

func (p *SecretPatch) parseTypeAndImmutable(doc map[string]json.RawMessage) error {
	if raw, found := doc["type"]; found {
		if isNull(raw) {
			return errors.New("type can't be null")
		}
		if err := json.Unmarshal(raw, &p.Type); err != nil {
			return fmt.Errorf("type: %w", err)
		}
	}
	if raw, found := doc["immutable"]; found {
		if isNull(raw) {
			return errors.New("immutable can't be null")
		}
		if err := json.Unmarshal(raw, &p.Immutable); err != nil {
			return fmt.Errorf("immutable: %w", err)
		}
	}
	return nil
}

// decodeObject decodes raw, which must be a JSON object, into v.
func decodeObject(raw json.RawMessage, member string, v any) error {
	if isNull(raw) {
		return fmt.Errorf("%s can't be null", member)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("%s: %w", member, err)
	}
	return nil
}

func isNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
