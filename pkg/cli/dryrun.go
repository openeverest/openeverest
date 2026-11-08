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

package cli

import "fmt"

// Values of --dry-run, named as kubectl names them.
const (
	DryRunNone   = "none"
	DryRunClient = "client" // print what would be sent, without sending it
	DryRunServer = "server" // send with dryRun=All: the server runs every check but persists nothing
)

// CheckDryRun rejects a --dry-run value other than none, client or server.
func CheckDryRun(mode string) error {
	switch mode {
	case "", DryRunNone, DryRunClient, DryRunServer:
		return nil
	}
	return fmt.Errorf("invalid --dry-run value %q: must be %q, %q or %q", mode, DryRunNone, DryRunClient, DryRunServer)
}

// IsDryRun reports whether mode previews the write instead of making it.
func IsDryRun(mode string) bool {
	return mode == DryRunClient || mode == DryRunServer
}
