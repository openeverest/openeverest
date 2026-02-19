// everest
// Copyright (C) 2025 Percona LLC
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

package session

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
)

func TestCleanupOld(t *testing.T) {
	type tcase struct {
		name          string
		data          string
		thresholdDate time.Time
		expected      []string
	}
	l := zaptest.NewLogger(t).Sugar()
	tcases := []tcase{
		{
			name:          "one outdated",
			data:          "id123AAA1743687192,id2323BBB1743687194",
			thresholdDate: time.Date(2025, 4, 3, 13, 33, 13, 0, time.UTC),
			expected:      []string{"id2323BBB1743687194"},
		},
		{
			name:          "two outdated",
			data:          "id123AAA1743687192,id2323BBB1743687193,id2323BBB1743687194",
			thresholdDate: time.Date(2025, 4, 3, 13, 33, 13, 0, time.UTC),
			expected:      []string{"id2323BBB1743687194"},
		},
		{
			name:          "all outdated",
			data:          "id123AAA1743687191,id2323BBB1743687192,id2323BBB1743687193",
			thresholdDate: time.Date(2025, 4, 3, 13, 33, 13, 0, time.UTC),
			expected:      []string{},
		},
		{
			name:          "all fresh",
			data:          "id123AAA1743687194,id2323BBB1743687195,id2323BBB1743687196",
			thresholdDate: time.Date(2025, 4, 3, 13, 33, 13, 0, time.UTC),
			expected:      []string{"id123AAA1743687194", "id2323BBB1743687195", "id2323BBB1743687196"},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, cleanupOld(l, tc.data, tc.thresholdDate))
		})
	}
}

func TestAddDataToSecret(t *testing.T) {
	type tcase struct {
		name          string
		data          string
		secret        *corev1.Secret
		thresholdDate time.Time
		expected      *corev1.Secret
	}
	l := zaptest.NewLogger(t).Sugar()
	tcases := []tcase{
		{
			name:          "empty secret",
			data:          "id123AAA1743687192",
			secret:        &corev1.Secret{},
			thresholdDate: time.Date(2025, 4, 3, 13, 33, 10, 0, time.UTC),
			expected: &corev1.Secret{
				StringData: map[string]string{
					dataKey: "id123AAA1743687192",
				},
			},
		},
		{
			name: "nothing to delete, add newer data",
			data: "id123AAA1743687192",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					dataKey: []byte("id123AAA1743687191"),
				},
			},
			thresholdDate: time.Date(2025, 4, 3, 13, 33, 10, 0, time.UTC),
			expected: &corev1.Secret{
				// the Data field is updated only when the object is applied to k8s, so for this test
				// only the write-only StringData field is expected to be changed.
				Data: map[string][]byte{
					dataKey: []byte("id123AAA1743687191"),
				},
				StringData: map[string]string{
					dataKey: "id123AAA1743687191,id123AAA1743687192",
				},
			},
		},
		{
			name: "deleted old data, add newer data",
			data: "id123AAA1743687194",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					dataKey: []byte("id123AAA1743687191,id123AAA1743687193"),
				},
			},
			thresholdDate: time.Date(2025, 4, 3, 13, 33, 12, 0, time.UTC),
			expected: &corev1.Secret{
				// the Data field is updated only when the object is applied to k8s, so for this test
				// only the write-only StringData field is expected to be changed.
				Data: map[string][]byte{
					dataKey: []byte("id123AAA1743687191,id123AAA1743687193"),
				},
				StringData: map[string]string{
					dataKey: "id123AAA1743687193,id123AAA1743687194",
				},
			},
		},
		{
			name: "deleted all old data, add newer data",
			data: "id123AAA1743687195",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					dataKey: []byte("id123AAA1743687191,id123AAA1743687193"),
				},
			},
			thresholdDate: time.Date(2025, 4, 3, 13, 33, 14, 0, time.UTC),
			expected: &corev1.Secret{
				// the Data field is updated only when the object is applied to k8s, so for this test
				// only the write-only StringData field is expected to be changed.
				Data: map[string][]byte{
					dataKey: []byte("id123AAA1743687191,id123AAA1743687193"),
				},
				StringData: map[string]string{
					dataKey: "id123AAA1743687195",
				},
			},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := addDataToSecret(l, tc.secret, tc.data, tc.thresholdDate)
			assert.Equal(t, tc.expected, result)
		})
	}
}

/*
This benchmark measures how much time does it take to clean up a long lists of tokens.
On an Apple M3 Pro it takes ~0.67 ms to perform a cleanup for a list of 10,000 tokens, which is acceptable.

goos: darwin
goarch: arm64
pkg: github.com/openeverest/openeverest/v2/pkg/session
cpu: Apple M3 Pro
BenchmarkCleanupOld
BenchmarkCleanupOld-12    	    1500	    671899 ns/op
*/
func BenchmarkCleanupOld(b *testing.B) {
	numTokens := 10000
	list := generateTestList(numTokens)
	thresholdDate := time.Date(2025, 4, 3, 13, 33, 1, 0, time.UTC)
	l := zap.L().Sugar()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cleanupOld(l, list, thresholdDate)
	}
}

func generateTestList(numTokens int) string {
	var builder strings.Builder
	expDate := time.Date(2070, 4, 3, 13, 33, 1, 0, time.UTC).Unix()
	for i := 0; i < numTokens; i++ {
		// expiration date is year 2070 which is long ahead, so all the tokens should be kept
		builder.WriteString("21669bd9-2374-4dc1-9238-77d5cad01fed" + fmt.Sprintf("%d", expDate))
		if i < numTokens-1 {
			builder.WriteString(sep)
		}
	}
	return builder.String()
}

const (
	stringSizeMB = 1 // Size of the string in MB
)

// generateLargeString creates a string of the specified size in MB.
func generateLargeString(sizeMB int) string {
	size := sizeMB * 1024 * 1024 // Convert MB to bytes
	builder := strings.Builder{}
	builder.Grow(size)

	// Repeat a simple pattern to fill the string. This is important.
	// Random data will compress differently and may not represent realistic
	// data patterns.
	pattern := "abcdefghijklmnopqrstuvwxyz"
	patternLen := len(pattern)

	for i := 0; i < size; i++ {
		builder.WriteByte(pattern[i%patternLen])
	}

	return builder.String()
}

// BenchmarkStringsContains measures the time it takes to run strings.Contains on a 1MB string.
func BenchmarkStringsContains(b *testing.B) {
	largeString := generateLargeString(stringSizeMB)
	// Substring to search for.  Choose a substring that's *likely* to be found for a realistic test.
	substring := "xyz"

	// Reset the timer to exclude the setup time.
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		strings.Contains(largeString, substring)
	}
}

// BenchmarkStringsContainsNotFound measures the time it takes to run strings.Contains on a 1MB string when the substring is NOT present.
func BenchmarkStringsContainsNotFound(b *testing.B) {
	largeString := generateLargeString(stringSizeMB)
	substring := "thisstringisdefinitelynotpresent" // Substring that is NOT present.

	b.ResetTimer() // Reset the timer to exclude the setup time.

	for i := 0; i < b.N; i++ {
		strings.Contains(largeString, substring)
	}
}
