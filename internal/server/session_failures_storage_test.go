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

package server

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// TestRateLimiterMemoryStore_ConcurrentAllowAndIncreaseTimeout guards against a data race
// between Allow reading Visitor.timeout and IncreaseTimeout writing it. Run with -race.
func TestRateLimiterMemoryStore_ConcurrentAllowAndIncreaseTimeout(t *testing.T) {
	t.Parallel()

	store := NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{
		Rate: rate.Limit(1000),
	})

	const identifier = "1.2.3.4"
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(2)

	// Register the visitor first: IncreaseTimeout silently returns when the
	// identifier has no visitor yet, so without this the writer goroutine can
	// drain every iteration without ever touching Visitor.timeout.
	_, _ = store.Allow(identifier)

	go func() {
		defer wg.Done()
		for range iterations {
			_, _ = store.Allow(identifier)
		}
	}()

	go func() {
		defer wg.Done()
		for range iterations {
			store.IncreaseTimeout(identifier)
		}
	}()

	wg.Wait()

	expectedTimeout := initialTimeout << iterations

	store.mutex.Lock()
	v, ok := store.visitors[identifier]
	store.mutex.Unlock()

	require.True(t, ok, "Visitor was removed during the run")
	assert.Equal(t, expectedTimeout, v.timeout)
}
