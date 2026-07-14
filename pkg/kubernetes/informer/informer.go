// everest
// Copyright (C) 2023 Percona LLC
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

// Package informer provides generic utilities to work with Kubernetes informers.
package informer

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Informer is a generic wrapper around the controller-runtime cache.
// It provides a minimalist interface for watching Kubernetes objects
// and triggering callbacks.
type Informer struct {
	cache cache.Cache

	opts          cache.Options
	cfg           *rest.Config
	eventHandlers toolscache.ResourceEventHandlerFuncs
	l             *zap.SugaredLogger
}

// OptionsFunc is a function that sets options for the informer.
type OptionsFunc func(*Informer)

// WithConfig sets the rest.Config for the Informer.
func WithConfig(cfg *rest.Config) OptionsFunc {
	return func(i *Informer) {
		i.cfg = cfg
	}
}

// WithLogger sets the logger for the Informer.
func WithLogger(l *zap.SugaredLogger) OptionsFunc {
	return func(i *Informer) {
		i.l = l
	}
}

// Watches sets the Informer to watch the given object.
// If a namespace is provided, the Informer will only watch the object only in
// that namespace.
func Watches(obj client.Object, ns ...string) OptionsFunc {
	return func(i *Informer) {
		if i.opts.ByObject == nil {
			i.opts.ByObject = make(map[client.Object]cache.ByObject)
		}
		i.opts.ByObject[obj] = cache.ByObject{}
		namespace := ""
		if len(ns) > 0 {
			namespace = ns[0]
		}
		if namespace != "" {
			i.opts.ByObject[obj] = cache.ByObject{
				Namespaces: map[string]cache.Config{
					namespace: {},
				},
			}
		}
	}
}

// New creates and returns a new Informer.
func New(opts ...OptionsFunc) (*Informer, error) {
	i := &Informer{}
	for _, opt := range opts {
		opt(i)
	}

	cache, err := cache.New(i.cfg, i.opts)
	if err != nil {
		return nil, err
	}
	i.cache = cache
	return i, nil
}

// OnUpdate is triggered when an object is updated.
func (i *Informer) OnUpdate(cb func(oldObj, newObj any)) {
	i.eventHandlers.UpdateFunc = cb
}

// OnAdd is triggered when an object is added.
func (i *Informer) OnAdd(cb func(obj any)) {
	i.eventHandlers.AddFunc = cb
}

// OnDelete is triggered when an object is deleted.
func (i *Informer) OnDelete(cb func(obj any)) {
	i.eventHandlers.DeleteFunc = cb
}

// Start the Informer.
func (i *Informer) Start(ctx context.Context, obj client.Object) error {
	// Get the Informer for the specified object.
	inf, err := i.cache.GetInformer(ctx, obj)
	if err != nil {
		return errors.Join(err, errors.New("failed to get Informer"))
	}
	// Register callbacks.
	if _, err := inf.AddEventHandler(i.eventHandlers); err != nil {
		return errors.Join(err, errors.New("failed to add event handler"))
	}
	// Start the cache in a separate goroutine, since it is a blocking call.
	go func() {
		if err := i.cache.Start(ctx); err != nil {
			i.l.Error("failed to start cache", zap.Error(err))
		}
	}()
	return nil
}
