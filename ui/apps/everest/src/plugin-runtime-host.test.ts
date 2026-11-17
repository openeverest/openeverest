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

import * as React from 'react';
import './plugin-runtime-host';

describe('plugin runtime host', () => {
  it('exposes the host React singleton', () => {
    expect(window.__EVEREST_PLUGIN_RUNTIME__?.React.useState).toBe(
      React.useState
    );
  });

  it('cannot be replaced or mutated by a plugin', () => {
    const runtime = window.__EVEREST_PLUGIN_RUNTIME__;

    expect(() => {
      window.__EVEREST_PLUGIN_RUNTIME__ = undefined;
    }).toThrow(TypeError);
    expect(
      Reflect.defineProperty(window, '__EVEREST_PLUGIN_RUNTIME__', {
        value: {},
      })
    ).toBe(false);
    expect(Reflect.set(runtime ?? {}, 'React', {})).toBe(false);
    expect(window.__EVEREST_PLUGIN_RUNTIME__).toBe(runtime);
    expect(window.__EVEREST_PLUGIN_RUNTIME__?.React.useState).toBe(
      React.useState
    );
  });
});
