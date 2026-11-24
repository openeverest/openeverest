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

	"github.com/blang/semver/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Reasons of a Result.
const (
	// ReasonSupported means the extension's stamp is inside the window.
	ReasonSupported = "Supported"
	// ReasonCoreTooOld means the extension was built against a newer core
	// than this one serves.
	ReasonCoreTooOld = "CoreTooOld"
	// ReasonExtensionTooOld means the extension was built against a core
	// older than the oldest this one still serves.
	ReasonExtensionTooOld = "ExtensionTooOld"
	// ReasonUnversioned means the extension carries no stamp.
	ReasonUnversioned = "Unversioned"
	// ReasonInvalidVersion means the stamp or the window is not a SemVer
	// 2.0.0 version.
	ReasonInvalidVersion = "InvalidVersion"
	// ReasonCoreVersionSkew means the parts of a core disagree on its
	// window, as happens while an upgrade is under way: Check reports it for
	// an empty window, the intersection of windows that do not overlap, and
	// core reports it while the window on its CRDs differs from the one it
	// was built with.
	ReasonCoreVersionSkew = "CoreVersionSkew"
	// ReasonCoreTooOldAcknowledged is ReasonCoreTooOld after a user chose
	// to run the extension anyway.
	ReasonCoreTooOldAcknowledged = "CoreTooOldAcknowledged"
	// ReasonExtensionTooOldAcknowledged is ReasonExtensionTooOld after a
	// user chose to run the extension anyway.
	ReasonExtensionTooOldAcknowledged = "ExtensionTooOldAcknowledged"
)

// Result is the outcome of Check, shaped like a condition: Status is True
// when the extension is supported, False when it is not, and Unknown when
// the versions cannot be compared.
type Result struct {
	Status metav1.ConditionStatus
	Reason string
}

// Check reports whether an extension stamped with ext can run on a core
// whose window is [minimum, current]. It compares by plain SemVer 2.0.0
// precedence, with no version ranges, so that every other implementation of
// the check can match it exactly; OrderVectors holds the cases they share.
// A window whose minimum is above its current version serves nothing. Only
// intersecting the windows of parts of a core that disagree produces one,
// so Check reports CoreVersionSkew for it rather than blaming the extension.
func Check(ext, minimum, current string) Result {
	switch {
	case ext == "":
		return Result{Status: metav1.ConditionUnknown, Reason: ReasonUnversioned}
	case !Valid(ext), !Valid(minimum), !Valid(current):
		return Result{Status: metav1.ConditionUnknown, Reason: ReasonInvalidVersion}
	case Compare(minimum, current) > 0:
		return Result{Status: metav1.ConditionUnknown, Reason: ReasonCoreVersionSkew}
	case Compare(current, ext) < 0:
		return Result{Status: metav1.ConditionFalse, Reason: ReasonCoreTooOld}
	case Compare(ext, minimum) < 0:
		return Result{Status: metav1.ConditionFalse, Reason: ReasonExtensionTooOld}
	default:
		return Result{Status: metav1.ConditionTrue, Reason: ReasonSupported}
	}
}

// Valid reports whether v is a SemVer 2.0.0 version: exactly
// MAJOR.MINOR.PATCH with optional pre-release and build metadata, no "v"
// prefix, no short forms such as 2.0, and no leading zeros.
func Valid(v string) bool {
	_, err := semver.Parse(v)
	return err == nil
}

// Compare returns -1, 0 or 1 as a sorts before, equal to or after b by
// SemVer 2.0.0 precedence, which ignores build metadata. Numeric pre-release
// identifiers compare as numbers and sort before alphanumeric ones, which
// compare as ASCII strings, so alpha < beta < dev < rc: a 2.0.0-dev.N build
// sorts above every 2.0.0 alpha and beta. Only values Valid accepts have a
// precedence; the rest sort before them, among themselves by bytes.
func Compare(a, b string) int {
	va, errA := semver.Parse(a)
	vb, errB := semver.Parse(b)
	switch {
	case errA != nil && errB != nil:
		return strings.Compare(a, b)
	case errA != nil:
		return -1
	case errB != nil:
		return 1
	default:
		return va.Compare(vb)
	}
}

// IsDevelopment reports whether v is a development value, the version core
// has between releases: a Valid version whose last pre-release identifier
// is next, such as 2.0.0-alpha.1.next, or whose pre-release is exactly
// alpha.0, such as 2.1.0-alpha.0. No release carries one, so a stamp with
// such a value names a core nobody can install. A pre-release of next
// alone, as in 2.0.0-next, is one too, so no release can be named that
// way either. Core dev builds such as 2.0.0-dev.4 are not development
// values: users can install them.
func IsDevelopment(v string) bool {
	parsed, err := semver.Parse(v)
	if err != nil || len(parsed.Pre) == 0 {
		return false
	}
	if last := parsed.Pre[len(parsed.Pre)-1]; !last.IsNum && last.VersionStr == "next" {
		return true
	}
	return len(parsed.Pre) == 2 &&
		!parsed.Pre[0].IsNum && parsed.Pre[0].VersionStr == "alpha" &&
		parsed.Pre[1].IsNum && parsed.Pre[1].VersionNum == 0
}

// Acknowledge returns r after a user chose to run the extension anyway. The
// status of an unsupported result stays False, so the acknowledgement shows
// without hiding that the extension is outside the window; every other
// result is returned unchanged.
func Acknowledge(r Result) Result {
	switch r.Reason {
	case ReasonCoreTooOld:
		r.Reason = ReasonCoreTooOldAcknowledged
	case ReasonExtensionTooOld:
		r.Reason = ReasonExtensionTooOldAcknowledged
	}
	return r
}

// Message describes r, the result of checking the stamp ext against w, to a
// user. It is a predicate to put after the name and version of the
// extension, as in "psmdb 0.8.0 needs OpenEverest 2.1.0 or later". It names
// core releases only: a development value, which nobody can install, is
// described as an unreleased OpenEverest build.
func Message(r Result, ext string, w Window) string {
	switch r.Reason {
	case ReasonSupported:
		return "works with this version of OpenEverest"
	case ReasonCoreTooOld:
		return coreTooOldMessage(ext)
	case ReasonExtensionTooOld:
		return extensionTooOldMessage(w)
	case ReasonCoreTooOldAcknowledged:
		return coreTooOldMessage(ext) + acknowledgedSuffix
	case ReasonExtensionTooOldAcknowledged:
		return extensionTooOldMessage(w) + acknowledgedSuffix
	case ReasonUnversioned:
		return "does not declare the OpenEverest it was built for, so its compatibility cannot be checked"
	case ReasonInvalidVersion:
		if !Valid(ext) {
			return fmt.Sprintf("declares an invalid OpenEverest version %q, so its compatibility cannot be checked", ext)
		}
		return "cannot be checked for compatibility because this OpenEverest publishes invalid supported versions"
	case ReasonCoreVersionSkew:
		return "cannot be checked for compatibility while the parts of this OpenEverest disagree on the versions they support, as they do during an upgrade"
	default:
		return fmt.Sprintf("cannot be checked for compatibility (%s)", r.Reason)
	}
}

const acknowledgedSuffix = "; allowed anyway because the incompatibility was acknowledged"

func coreTooOldMessage(ext string) string {
	if IsDevelopment(ext) {
		return "needs a newer, unreleased OpenEverest build; install a version of it built for a released OpenEverest"
	}
	return fmt.Sprintf("needs OpenEverest %s or later", ext)
}

func extensionTooOldMessage(w Window) string {
	if IsDevelopment(w.Min) {
		return "was built for an older OpenEverest; update it to a version built for this unreleased OpenEverest build"
	}
	return fmt.Sprintf("was built for an older OpenEverest; update it to a release built for OpenEverest %s or later", w.Min)
}
