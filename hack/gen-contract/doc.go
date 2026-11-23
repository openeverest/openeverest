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

// Command gen-contract works out the contract version of this core and the
// window of contract versions it serves, and writes them to the contract
// package: VERSION and MIN_VERSION, which tools that do not build Go, such
// as release workflows, read from any git ref, and zz_generated.version.go.
//
// Nobody picks the version. The command fingerprints the contract artifacts
// (see inputs) and compares the fingerprint with the one recorded for the
// latest release in contract/releases: while they are equal the version is
// that release's contract version, and once they differ it is the
// development value after that release. The minimum moves only at a
// release.
//
// With -check-release it only checks that the contract is the one the
// latest release was cut with, as a release branch must.
//
// It imports nothing from this module, so it runs whatever state the
// contract package is in, the file it generates included.
package main
