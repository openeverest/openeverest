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

package contract

import (
	"fmt"
	"strings"
)

// AcknowledgementHolds reports whether an acknowledgement naming the core
// release ack still applies to a core whose CRDs publish contractVersion,
// the VersionAnnotation, and coreVersion, the CoreVersionKey label. It holds
// while
//
//	contractVersion <= ack <= coreVersion
//
// by the precedence of Compare, and is false when any value is not Valid.
//
// A user who acknowledges an incompatibility accepts what the extension does
// with the contract the core serves, so the acknowledgement lasts as long as
// that contract. It survives the patch releases that keep the contract,
// which would otherwise teach admins to acknowledge again without reading,
// and expires with the first release that changes it, which is exactly when
// what the user accepted changes. The upper bound means an acknowledgement
// written ahead of time, in Git before an upgrade for example, does nothing
// until that release is installed: users acknowledge a core they can see.
func AcknowledgementHolds(ack, contractVersion, coreVersion string) bool {
	if !Valid(ack) || !Valid(contractVersion) || !Valid(coreVersion) {
		return false
	}
	return Compare(contractVersion, ack) <= 0 && Compare(ack, coreVersion) <= 0
}

// ParseAcknowledgedExtensions reads the value of
// AcknowledgedExtensionsAnnotation, comma-separated <name>@<release>
// entries, into a map from the name of an extension's root object to the
// core release it is acknowledged on. Spaces around entries, names and
// releases are ignored, and a value with nothing but spaces has no entries.
//
// It returns an error for an entry without a name, a release that is not
// Valid, or a name listed more than once, whose releases would leave it
// unclear which one applies. A list that does not parse acknowledges
// nothing: readers must not guess which part of it the admin meant.
func ParseAcknowledgedExtensions(v string) (map[string]string, error) {
	acks := make(map[string]string)
	if strings.TrimSpace(v) == "" {
		return acks, nil
	}
	for entry := range strings.SplitSeq(v, ",") {
		name, release, found := strings.Cut(entry, "@")
		name, release = strings.TrimSpace(name), strings.TrimSpace(release)
		switch {
		case !found:
			return nil, fmt.Errorf("acknowledged extension %q is not <name>@<release>", entry)
		case name == "":
			return nil, fmt.Errorf("acknowledged extension %q has no name", entry)
		case !Valid(release):
			return nil, fmt.Errorf("acknowledged extension %q: %q is not a SemVer 2.0.0 version", name, release)
		}
		if _, ok := acks[name]; ok {
			return nil, fmt.Errorf("acknowledged extension %q is listed more than once", name)
		}
		acks[name] = release
	}
	return acks, nil
}
