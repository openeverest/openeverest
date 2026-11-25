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

package summary

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	contractsummary "github.com/openeverest/openeverest/v2/contract/summary"
)

// CheckConditions returns every way the summary conditions break the rules
// Compute writes them by, or nil when they keep them all. It moved to
// contract/summary, which core imports too, so that core no longer imports
// the provider runtime.
//
// Deprecated: use contract/summary.CheckConditions; removed in the next
// release.
func CheckConditions(conds []metav1.Condition) error {
	return contractsummary.CheckConditions(conds)
}
