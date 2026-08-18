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
	"time"

	"golang.org/x/time/rate"
)

const (
	initialTimeout         = 20 * time.Millisecond
	attemptsExpiration     = 6 * time.Hour
	defaultStoreExpiration = 3 * time.Minute
)

// RateLimiterMemoryStore is the built-in store implementation for RateLimiter.
type RateLimiterMemoryStore struct {
	visitors map[string]*Visitor
	mutex    sync.Mutex
	rate     rate.Limit // for more info check out Limiter docs - https://pkg.go.dev/golang.org/x/time/rate#Limit.

	burst       int
	expiresIn   time.Duration
	lastCleanup time.Time

	timeNow func() time.Time
}

// Visitor signifies a unique user's limiter details.
type Visitor struct {
	*rate.Limiter

	lastSeen time.Time
	timeout  time.Duration
	expire   time.Duration
}

// NewRateLimiterMemoryStoreWithConfig creates an in-memory storage which can be used in echomiddleware.RateLimiterWithConfig.
func NewRateLimiterMemoryStoreWithConfig(config RateLimiterMemoryStoreConfig) *RateLimiterMemoryStore {
	store := &RateLimiterMemoryStore{}

	store.rate = config.Rate
	store.burst = config.Burst
	store.expiresIn = config.ExpiresIn
	if config.ExpiresIn == 0 {
		store.expiresIn = defaultStoreExpiration
	}
	if config.Burst == 0 {
		store.burst = int(config.Rate)
	}
	store.visitors = make(map[string]*Visitor)
	store.timeNow = time.Now
	store.lastCleanup = store.timeNow()
	return store
}

// RateLimiterMemoryStoreConfig represents configuration for RateLimiterMemoryStore.
type RateLimiterMemoryStoreConfig struct {
	Rate      rate.Limit    // Rate of requests allowed to pass as req/s. For more info check out Limiter docs - https://pkg.go.dev/golang.org/x/time/rate#Limit.
	Burst     int           // Burst is maximum number of requests to pass at the same moment. It additionally allows a number of requests to pass when rate limit is reached.
	ExpiresIn time.Duration // ExpiresIn is the duration after that a rate limiter is cleaned up
}

// Allow implements RateLimiterStore.Allow
// checks if the request doesn't violate the configured limits per second and the individual timeout in case of failures.
func (store *RateLimiterMemoryStore) Allow(identifier string) (bool, error) {
	store.mutex.Lock()
	limiter, exists := store.visitors[identifier]
	if !exists {
		limiter = &Visitor{}
		limiter.Limiter = rate.NewLimiter(store.rate, store.burst)
		limiter.expire = attemptsExpiration
		limiter.timeout = initialTimeout
		store.visitors[identifier] = limiter
	}
	now := store.timeNow()
	sinceLastFailure := now.Sub(limiter.lastSeen)
	timeout := limiter.timeout

	limiter.lastSeen = now
	if now.Sub(store.lastCleanup) > store.expiresIn {
		store.cleanupStaleVisitors()
	}
	store.mutex.Unlock()
	return limiter.AllowN(store.timeNow(), 1) && sinceLastFailure > timeout, nil
}

// CleanupVisitor removes the data about previous failures. To be used when a successful opetation was performed.
func (store *RateLimiterMemoryStore) CleanupVisitor(identifier string) {
	store.mutex.Lock()
	delete(store.visitors, identifier)
	store.mutex.Unlock()
}

// IncreaseTimeout increases timeout for the given visitor.
func (store *RateLimiterMemoryStore) IncreaseTimeout(identifier string) {
	store.mutex.Lock()
	limiter, exists := store.visitors[identifier]
	if exists {
		limiter.timeout *= 2
	}
	store.mutex.Unlock()
}

/*
cleanupStaleVisitors helps manage the size of the visitors map by removing stale records
of users who haven't visited again after the configured expiry time has elapsed.
*/
func (store *RateLimiterMemoryStore) cleanupStaleVisitors() {
	for id, visitor := range store.visitors {
		if store.timeNow().Sub(visitor.lastSeen) > visitor.expire {
			delete(store.visitors, id)
		}
	}
	store.lastCleanup = store.timeNow()
}
