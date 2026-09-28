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

// Exposes the host's React runtime as a global so dynamically imported plugin
// bundles resolve `react`/`react-dom` to the host copy through the import map
// in index.html. A plugin that bundles its own MUI must still share the host
// React, otherwise it loads a second React instance and hooks/theme context break.
import * as React from 'react';
import * as ReactDOM from 'react-dom';
import * as ReactJSXRuntime from 'react/jsx-runtime';

declare global {
  interface Window {
    __EVEREST_PLUGIN_RUNTIME__?: {
      React: typeof React;
      ReactDOM: typeof ReactDOM;
      ReactJSXRuntime: typeof ReactJSXRuntime;
    };
  }
}

// Read-only and non-replaceable so one plugin (or injected script) can't swap
// the React every other plugin resolves through the import map.
Object.defineProperty(window, '__EVEREST_PLUGIN_RUNTIME__', {
  value: Object.freeze({
    React,
    ReactDOM,
    ReactJSXRuntime,
  }),
  writable: false,
  configurable: false,
  enumerable: true,
});
