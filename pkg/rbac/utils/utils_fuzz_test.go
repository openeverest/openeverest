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

package utils

import (
	"testing"

	"github.com/casbin/casbin/v2/model"
)

// fuzzRBACModel is a minimal Casbin RBAC model definition sufficient to
// exercise LoadPolicyLine's "p" and "g" line handling.
const fuzzRBACModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
`

func FuzzLoadPolicyLine(f *testing.F) {
	seeds := []string{
		"",
		"#",
		"# comment",
		"p, alice, data1, read",
		"g, alice, admin",
		"p, alice, data1",
		"p",
		"g",
		"x, alice, data1, read",
		"p, \"alice\", data1, read",
		"p, alice, data1, read, extra, fields",
		"p,,,",
		",,,,",
		"\"",
		"p, alice, data1, \"unterminated",
		"p, alice, /namespaces/*/databases/*, read",
		"\x00\x01\x02",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, line string) {
		m, err := model.NewModelFromString(fuzzRBACModel)
		if err != nil {
			t.Fatalf("failed to build fuzz model: %v", err)
		}
		// We only assert that LoadPolicyLine does not panic. Returning an
		// error for malformed input is the expected and correct behavior.
		_ = LoadPolicyLine(line, m)
	})
}
