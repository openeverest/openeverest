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

import { ReactNode } from 'react';
import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { useDevMode, DEV_MODE_STORAGE_KEY } from './useDevMode';

const wrapperAt = (path: string) => {
  const Wrapper = ({ children }: { children: ReactNode }) => (
    <MemoryRouter initialEntries={[path]}>{children}</MemoryRouter>
  );
  return Wrapper;
};

describe('useDevMode', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('defaults to false with no flag and no query param', () => {
    const { result } = renderHook(() => useDevMode(), {
      wrapper: wrapperAt('/databases'),
    });

    expect(result.current).toBe(false);
  });

  it('turns on and persists when a page is opened with ?dev=true', () => {
    const { result } = renderHook(() => useDevMode(), {
      wrapper: wrapperAt('/databases?dev=true'),
    });

    expect(result.current).toBe(true);
    expect(localStorage.getItem(DEV_MODE_STORAGE_KEY)).toBe('true');
  });

  it('stays on from persisted storage even without the query param', () => {
    localStorage.setItem(DEV_MODE_STORAGE_KEY, 'true');

    const { result } = renderHook(() => useDevMode(), {
      wrapper: wrapperAt('/settings'),
    });

    expect(result.current).toBe(true);
  });

  it('turns back off when a page is opened with ?dev=false', () => {
    localStorage.setItem(DEV_MODE_STORAGE_KEY, 'true');

    const { result } = renderHook(() => useDevMode(), {
      wrapper: wrapperAt('/databases?dev=false'),
    });

    expect(result.current).toBe(false);
    expect(localStorage.getItem(DEV_MODE_STORAGE_KEY)).toBe('false');
  });

  it('ignores an unrelated dev value and keeps the stored state', () => {
    localStorage.setItem(DEV_MODE_STORAGE_KEY, 'true');

    const { result } = renderHook(() => useDevMode(), {
      wrapper: wrapperAt('/databases?dev=maybe'),
    });

    expect(result.current).toBe(true);
    expect(localStorage.getItem(DEV_MODE_STORAGE_KEY)).toBe('true');
  });
});
