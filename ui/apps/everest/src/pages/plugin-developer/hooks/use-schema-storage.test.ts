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

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useSchemaStorage } from './use-schema-storage';

const DRAFT_KEY = 'everest.playground.draft';
const SAVED_KEY = 'everest.playground.saved';
const FALLBACK = 'fallback: yaml';

beforeEach(() => localStorage.clear());
afterEach(() => vi.restoreAllMocks());

describe('useSchemaStorage', () => {
  it('returns the fallback draft when nothing is stored', () => {
    const { result } = renderHook(() => useSchemaStorage(FALLBACK));

    expect(result.current.initialDraft).toBe(FALLBACK);
  });

  it('returns the stored draft when one is present', () => {
    localStorage.setItem(DRAFT_KEY, JSON.stringify('stored: draft'));

    const { result } = renderHook(() => useSchemaStorage(FALLBACK));

    expect(result.current.initialDraft).toBe('stored: draft');
  });

  it('falls back when the stored draft cannot be parsed', () => {
    localStorage.setItem(DRAFT_KEY, '{not valid json');

    const { result } = renderHook(() => useSchemaStorage(FALLBACK));

    expect(result.current.initialDraft).toBe(FALLBACK);
  });

  it('roundtrips a saved schema and exposes its name sorted', () => {
    const { result } = renderHook(() => useSchemaStorage(FALLBACK));

    act(() => {
      result.current.saveSchema('beta', 'beta: yaml');
      result.current.saveSchema('alpha', 'alpha: yaml');
    });

    expect(result.current.names).toEqual(['alpha', 'beta']);
    expect(result.current.loadSchema('alpha')).toBe('alpha: yaml');
    expect(result.current.loadSchema('beta')).toBe('beta: yaml');
    expect(result.current.loadSchema('missing')).toBeUndefined();
  });

  it('overwrites a schema saved again under the same name', () => {
    const { result } = renderHook(() => useSchemaStorage(FALLBACK));

    act(() => result.current.saveSchema('one', 'first'));
    act(() => result.current.saveSchema('one', 'second'));

    expect(result.current.names).toEqual(['one']);
    expect(result.current.loadSchema('one')).toBe('second');
  });

  it('restores saved schemas from localStorage on mount', () => {
    localStorage.setItem(
      SAVED_KEY,
      JSON.stringify({ zebra: 'z: yaml', apple: 'a: yaml' })
    );

    const { result } = renderHook(() => useSchemaStorage(FALLBACK));

    expect(result.current.names).toEqual(['apple', 'zebra']);
    expect(result.current.loadSchema('zebra')).toBe('z: yaml');
  });

  it('deletes a schema and updates names', () => {
    const { result } = renderHook(() => useSchemaStorage(FALLBACK));

    act(() => {
      result.current.saveSchema('keep', 'k: yaml');
      result.current.saveSchema('drop', 'd: yaml');
    });
    expect(result.current.names).toEqual(['drop', 'keep']);

    act(() => result.current.deleteSchema('drop'));

    expect(result.current.names).toEqual(['keep']);
    expect(result.current.loadSchema('drop')).toBeUndefined();
  });

  it('persists a saved schema so a fresh mount can read it', () => {
    const first = renderHook(() => useSchemaStorage(FALLBACK));
    act(() => first.result.current.saveSchema('persisted', 'p: yaml'));

    const second = renderHook(() => useSchemaStorage(FALLBACK));
    expect(second.result.current.loadSchema('persisted')).toBe('p: yaml');
  });

  it('writes the draft to localStorage only after the debounce elapses', () => {
    vi.useFakeTimers();
    try {
      const { result } = renderHook(() => useSchemaStorage(FALLBACK));

      act(() => result.current.saveDraft('typed: yaml'));
      expect(localStorage.getItem(DRAFT_KEY)).toBeNull();

      act(() => vi.advanceTimersByTime(500));
      expect(localStorage.getItem(DRAFT_KEY)).toBe(
        JSON.stringify('typed: yaml')
      );
    } finally {
      vi.useRealTimers();
    }
  });

  it('debounces repeated draft writes to the last value', () => {
    vi.useFakeTimers();
    try {
      const { result } = renderHook(() => useSchemaStorage(FALLBACK));

      act(() => result.current.saveDraft('first'));
      act(() => vi.advanceTimersByTime(200));
      act(() => result.current.saveDraft('second'));
      act(() => vi.advanceTimersByTime(200));
      expect(localStorage.getItem(DRAFT_KEY)).toBeNull();

      act(() => vi.advanceTimersByTime(300));
      expect(localStorage.getItem(DRAFT_KEY)).toBe(JSON.stringify('second'));
    } finally {
      vi.useRealTimers();
    }
  });

  it('flushes a pending draft when the hook unmounts', () => {
    const { result, unmount } = renderHook(() => useSchemaStorage(FALLBACK));

    act(() => result.current.saveDraft('half typed'));
    // Still inside the debounce window, so nothing is written yet.
    expect(localStorage.getItem(DRAFT_KEY)).toBeNull();

    unmount();

    expect(JSON.parse(localStorage.getItem(DRAFT_KEY)!)).toBe('half typed');
  });

  it('does not throw when localStorage.setItem is unavailable', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('quota');
    });

    const { result } = renderHook(() => useSchemaStorage(FALLBACK));

    expect(() => {
      act(() => result.current.saveSchema('name', 'yaml'));
    }).not.toThrow();

    vi.useFakeTimers();
    try {
      expect(() => {
        act(() => result.current.saveDraft('yaml'));
        act(() => vi.advanceTimersByTime(500));
      }).not.toThrow();
    } finally {
      vi.useRealTimers();
    }
  });
});
