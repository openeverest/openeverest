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

package validation

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openeverest/openeverest/v2/pkg/events"
)

// validateMergePatch rejects a merge patch naming a member no caller may write,
// and returns the decoded document so a caller with its own resource-specific
// rules is not left decoding the same bytes a second time.
// A merge patch can address any member of the stored object, so what the typed
// verbs never expose has to be refused here instead.
func validateMergePatch(patch []byte) (map[string]any, error) {
	var doc map[string]any

	if err := json.Unmarshal(patch, &doc); err != nil || doc == nil {
		return nil, errors.Join(ErrInvalidRequest, errors.New("patch must be a JSON object"))
	}

	if _, found := doc["status"]; found {
		return nil, errors.Join(ErrInvalidRequest, errors.New("status may not be patched"))
	}
	if metadata, isObject := doc["metadata"].(map[string]any); isObject {
		if err := rejectProtectedMetadata(metadata); err != nil {
			return nil, errors.Join(ErrInvalidRequest, err)
		}
	}
	return doc, nil
}

// rejectProtectedMetadata reports the metadata members a patch may not name.
func rejectProtectedMetadata(metadata map[string]any) error {
	for _, member := range []string{"ownerReferences", "finalizers", "name", "namespace"} {
		if _, found := metadata[member]; found {
			return fmt.Errorf("metadata.%s may not be patched", member)
		}
	}

	annotations, found := metadata["annotations"]
	if !found {
		return nil
	}
	// Honouring a wipe and stamping the actor in one document is not expressible
	// in a merge patch, so the caller is pointed at the per-key route instead.
	if annotations == nil {
		return errors.New("metadata.annotations may not be removed wholesale; set individual keys to null instead")
	}
	named, isObject := annotations.(map[string]any)
	if !isObject {
		return nil
	}
	// The typed verbs stamp these after decoding, so a caller cannot author
	// them there. A merge patch is relayed as sent, so it could.
	for _, key := range []string{events.AnnotationActorType, events.AnnotationActorID} {
		if _, found := named[key]; found {
			return fmt.Errorf("annotation %s may not be patched", key)
		}
	}
	return nil
}
