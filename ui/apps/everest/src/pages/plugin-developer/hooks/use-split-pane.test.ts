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

import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { MutableRefObject } from 'react';
import { useSplitPane } from './use-split-pane';

// 1000px-wide container so clientX maps 1:1 to a percentage (500 -> 50%).
const attachContainer = (ref: MutableRefObject<HTMLDivElement | null>) => {
  const el = document.createElement('div');
  el.getBoundingClientRect = () => ({ left: 0, width: 1000 }) as DOMRect;
  ref.current = el;
};

const mouseMoveTo = (clientX: number) =>
  act(() => {
    document.dispatchEvent(new MouseEvent('mousemove', { clientX }));
  });

describe('useSplitPane', () => {
  it('starts at the initial width and not dragging', () => {
    const { result } = renderHook(() => useSplitPane({ initialLeftWidth: 30 }));

    expect(result.current.leftWidth).toBe(30);
    expect(result.current.isDragging).toBe(false);
  });

  it('ignores pointer movement until dragging starts', () => {
    const { result } = renderHook(() => useSplitPane());
    attachContainer(
      result.current.containerRef as MutableRefObject<HTMLDivElement | null>
    );

    mouseMoveTo(500);
    expect(result.current.leftWidth).toBe(25); // default, unchanged
  });

  it('resizes to the pointer position while dragging', () => {
    const { result } = renderHook(() => useSplitPane());
    attachContainer(
      result.current.containerRef as MutableRefObject<HTMLDivElement | null>
    );

    act(() => result.current.startDragging());
    expect(result.current.isDragging).toBe(true);

    mouseMoveTo(500);
    expect(result.current.leftWidth).toBe(50);
  });

  it('clamps movement to the min/max bounds', () => {
    const { result } = renderHook(() =>
      useSplitPane({ minLeftWidth: 20, maxLeftWidth: 80 })
    );
    attachContainer(
      result.current.containerRef as MutableRefObject<HTMLDivElement | null>
    );
    act(() => result.current.startDragging());

    mouseMoveTo(900); // 90% > max -> rejected
    expect(result.current.leftWidth).toBe(25);

    mouseMoveTo(100); // 10% < min -> rejected
    expect(result.current.leftWidth).toBe(25);

    mouseMoveTo(600); // 60% -> accepted
    expect(result.current.leftWidth).toBe(60);
  });

  it('stops dragging and detaches listeners on mouse up', () => {
    const { result } = renderHook(() => useSplitPane());
    attachContainer(
      result.current.containerRef as MutableRefObject<HTMLDivElement | null>
    );
    act(() => result.current.startDragging());

    act(() => {
      document.dispatchEvent(new MouseEvent('mouseup'));
    });
    expect(result.current.isDragging).toBe(false);

    // Listeners are torn down, so further movement is ignored.
    mouseMoveTo(700);
    expect(result.current.leftWidth).toBe(25);
  });
});
