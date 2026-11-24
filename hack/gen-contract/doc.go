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
// Nobody picks the version. The command computes a digest of each input of
// the contract (see inputs) under the current fingerprint scheme, and
// compares them with the digests the latest release record in
// contract/releases holds under that scheme: while every input the record
// holds is the same, the version is that release's contract version, and
// once one differs it is the development value after that release. The
// minimum moves only at a release. It fails, rather than guess, when the
// latest record has no digests under the current scheme; the seed record
// is exempt by name. It then stamps the window on the manifests core ships
// (see stamps), after the generators that write them.
//
// With -check-release it only checks that a release of the line it is
// given keeps the contract, when that release is a patch release of the
// latest one, as a release branch must. Its commands:
//
//	rebaseline     add the digests of the current scheme, computed from the
//	               tag of a release, to its record
//	diff           show how the canonical form of each input changed since
//	               a ref, or print the bytes a digest hashes
//	verify         check that the latest record holds the digests this
//	               generator computes from the tag of its release
//	check-records  check that release records are frozen
//	latest         print the latest release recorded
//	show           print a fact of the contract, such as its version, the
//	               scheme or whether a version is a development value,
//	               for tools that do not build Go (see runShow)
//	release        record a core release in contract/releases, with the
//	               fingerprints of its contract and the window and upgrade
//	               path the person who cuts it chooses; the release
//	               workflow runs it, then the command itself, before it
//	               tags the release:
//
//	gen-contract release -version 2.1.0 -type upgradable -min 2.0.0 -upgradable-from 2.0.0,2.0.1
//
// Commands that read a tag use git, and none fetches: make gen never needs
// the network. It imports nothing from this module, so it runs whatever
// state the contract package is in, the file it generates included.
package main
