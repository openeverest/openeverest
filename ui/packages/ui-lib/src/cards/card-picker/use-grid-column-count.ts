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

import { RefObject, useLayoutEffect, useState } from 'react';

// Number of columns that fit at the current width for a
// `repeat(auto-fit, minmax(minCardPx, 1fr))` grid. rAF-throttled so a resize
// drag doesn't thrash renders, and it only updates state when the count changes.
export const useGridColumnCount = (
  ref: RefObject<HTMLElement>,
  minCardPx: number,
  gapPx: number
): number => {
  const [columns, setColumns] = useState(1);

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;

    const apply = () =>
      setColumns((prev) => {
        const next = Math.max(
          1,
          Math.floor((el.clientWidth + gapPx) / (minCardPx + gapPx))
        );
        return prev === next ? prev : next;
      });

    // Measure synchronously on mount (before paint) to avoid a first-frame
    // flash; throttle only the subsequent resize callbacks.
    apply();

    let frame = 0;
    const onResize = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(apply);
    };
    const observer = new ResizeObserver(onResize);
    observer.observe(el);
    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
    };
  }, [ref, minCardPx, gapPx]);

  return columns;
};
