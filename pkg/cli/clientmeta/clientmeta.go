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

// Package clientmeta provides helpers for working with the Kubernetes object
// metadata on generated API client models. The generated types have no
// methods, so helpers that need per-item metadata access take an accessor
// closure instead of an interface constraint.
package clientmeta

import (
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SortByRecency orders items newest-first by metadata.creationTimestamp,
// falling back to name for stability when timestamps tie. Items without
// metadata or a timestamp sort last.
func SortByRecency[T any](items []T, meta func(*T) *metav1.ObjectMeta) {
	sort.Slice(items, func(i, j int) bool {
		mi, mj := meta(&items[i]), meta(&items[j])
		ti, iok := creationTime(mi)
		tj, jok := creationTime(mj)
		switch {
		case iok && jok && !ti.Equal(tj):
			return ti.After(tj)
		case iok != jok:
			return iok
		default:
			return name(mi) < name(mj)
		}
	})
}

func creationTime(md *metav1.ObjectMeta) (time.Time, bool) {
	if md == nil || md.CreationTimestamp.IsZero() {
		return time.Time{}, false
	}
	return md.CreationTimestamp.Time, true
}

func name(md *metav1.ObjectMeta) string {
	if md == nil {
		return ""
	}
	return md.Name
}
