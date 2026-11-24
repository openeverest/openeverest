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

// Package contract holds what OpenEverest core and its extensions agree on,
// and the check that decides whether an extension can run on a core.
//
// The contract is everything core and extensions share: the CRDs, the HTTP
// API, the webhook conventions, and the Go identifiers and rules both sides
// use. It has a single version, Version: the SemVer of the last core release
// that changed it. It is a core release number rather than a counter of its
// own because that is the number users and extension authors already know.
// An extension built against core 2.1.0 needs "OpenEverest 2.1.0 or later",
// with no table to map one numbering onto the other, and a patch release,
// which never changes the contract, keeps the version of the release that
// last did.
//
// A core serves extensions built for any version in its window
// [MinVersion, Version]. Extensions never choose a number: their tooling
// stamps every build with the Version of the core it was built against, as
// the VersionAnnotation on the objects core interprets and in the
// annotations of the extension's Chart.yaml. Check compares a stamp with a
// window by plain SemVer 2.0.0 precedence. There are no version ranges: a
// range an author writes is a guess about cores that do not exist yet,
// while the stamp is a fact about the build and the window is the core's own
// statement of what it serves.
//
// Between releases Version holds a development value that sorts after the
// last release and before any release that can follow it, so nobody has to
// guess the name of the next release: R.dev after a pre-release R (for
// example 2.0.0-alpha.1.dev) and X.Y.(Z+1)-dev after a release X.Y.Z (for
// example 2.0.1-dev). An extension stamped with a development value was
// built against an unreleased core and is not meant to be published.
//
// A user can run an extension outside the window anyway by acknowledging the
// incompatibility on a core release. The acknowledgement holds until a core
// release changes the contract, and does nothing before the release it names
// is installed; AcknowledgementHolds decides it from values the core CRDs
// publish.
//
// OrderVectors holds the cases that pin the ordering, the accepted syntax,
// the results of Check and when an acknowledgement holds, so that every
// other implementation, such as Helm templates or the UI, runs the same
// cases.
package contract
