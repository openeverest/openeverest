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

import { useState, useEffect, useRef, RefObject } from 'react';

type SplitPaneOptions = {
  initialLeftWidth?: number;
  minLeftWidth?: number;
  maxLeftWidth?: number;
};

type SplitPane = {
  containerRef: RefObject<HTMLDivElement>;
  leftWidth: number;
  isDragging: boolean;
  startDragging: () => void;
};

// Horizontal drag-to-resize for a two-pane layout. Widths are percentages of
// the container, clamped so neither pane can be dragged away entirely.
export const useSplitPane = ({
  initialLeftWidth = 25,
  minLeftWidth = 20,
  maxLeftWidth = 80,
}: SplitPaneOptions = {}): SplitPane => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [leftWidth, setLeftWidth] = useState(initialLeftWidth);
  const [isDragging, setIsDragging] = useState(false);

  useEffect(() => {
    if (!isDragging) return;

    const handleMouseMove = (e: MouseEvent) => {
      if (!containerRef.current) return;

      const rect = containerRef.current.getBoundingClientRect();
      const newLeftWidth = ((e.clientX - rect.left) / rect.width) * 100;

      if (newLeftWidth >= minLeftWidth && newLeftWidth <= maxLeftWidth) {
        setLeftWidth(newLeftWidth);
      }
    };

    const handleMouseUp = () => {
      setIsDragging(false);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isDragging, minLeftWidth, maxLeftWidth]);

  return {
    containerRef,
    leftWidth,
    isDragging,
    startDragging: () => setIsDragging(true),
  };
};
