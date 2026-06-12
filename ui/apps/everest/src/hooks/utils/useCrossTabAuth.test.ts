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

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useCrossTabAuth } from './useCrossTabAuth';

type MockChannel = {
  postMessage: ReturnType<typeof vi.fn>;
  close: ReturnType<typeof vi.fn>;
  onmessage: ((event: MessageEvent) => void) | null;
};

describe('useCrossTabAuth', () => {
  let mockChannel: MockChannel;

  beforeEach(() => {
    mockChannel = { postMessage: vi.fn(), close: vi.fn(), onmessage: null };
    vi.stubGlobal('BroadcastChannel', vi.fn(() => mockChannel));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('opens a BroadcastChannel on mount and closes it on unmount', () => {
    const { unmount } = renderHook(() => useCrossTabAuth(vi.fn()));
    expect(BroadcastChannel).toHaveBeenCalledWith('everest-auth');
    unmount();
    expect(mockChannel.close).toHaveBeenCalledOnce();
  });

  it('broadcastLogout posts a LOGOUT message to the channel', () => {
    const { result } = renderHook(() => useCrossTabAuth(vi.fn()));
    act(() => {
      result.current.broadcastLogout();
    });
    expect(mockChannel.postMessage).toHaveBeenCalledWith({ type: 'LOGOUT' });
  });

  it('calls onRemoteLogout when a LOGOUT message arrives from another tab', () => {
    const onRemoteLogout = vi.fn();
    renderHook(() => useCrossTabAuth(onRemoteLogout));

    act(() => {
      mockChannel.onmessage?.({ data: { type: 'LOGOUT' } } as MessageEvent);
    });

    expect(onRemoteLogout).toHaveBeenCalledOnce();
  });

  it('ignores messages with unknown types', () => {
    const onRemoteLogout = vi.fn();
    renderHook(() => useCrossTabAuth(onRemoteLogout));

    act(() => {
      mockChannel.onmessage?.({ data: { type: 'OTHER' } } as MessageEvent);
    });

    expect(onRemoteLogout).not.toHaveBeenCalled();
  });

  it('uses the latest callback without recreating the channel on rerender', () => {
    const firstCallback = vi.fn();
    const secondCallback = vi.fn();

    const { rerender } = renderHook(({ cb }) => useCrossTabAuth(cb), {
      initialProps: { cb: firstCallback },
    });

    expect(BroadcastChannel).toHaveBeenCalledTimes(1);

    rerender({ cb: secondCallback });

    expect(BroadcastChannel).toHaveBeenCalledTimes(1);

    act(() => {
      mockChannel.onmessage?.({ data: { type: 'LOGOUT' } } as MessageEvent);
    });

    expect(firstCallback).not.toHaveBeenCalled();
    expect(secondCallback).toHaveBeenCalledOnce();
  });

  it('does not call broadcastLogout after unmount', () => {
    const { result, unmount } = renderHook(() => useCrossTabAuth(vi.fn()));
    unmount();
    act(() => {
      result.current.broadcastLogout();
    });
    expect(mockChannel.postMessage).not.toHaveBeenCalled();
  });
});
