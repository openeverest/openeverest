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

import { useCallback, useEffect, useRef } from 'react';

const CHANNEL_NAME = 'everest-auth';

type AuthMessage = { type: 'LOGOUT' };

export const useCrossTabAuth = (onRemoteLogout: () => void) => {
  const channelRef = useRef<BroadcastChannel | null>(null);
  const callbackRef = useRef(onRemoteLogout);

  useEffect(() => {
    callbackRef.current = onRemoteLogout;
  });

  useEffect(() => {
    const channel = new BroadcastChannel(CHANNEL_NAME);
    channelRef.current = channel;

    channel.onmessage = ({ data }: MessageEvent<AuthMessage>) => {
      if (data.type === 'LOGOUT') {
        callbackRef.current();
      }
    };

    return () => {
      channel.close();
      channelRef.current = null;
    };
  }, []);

  const broadcastLogout = useCallback(() => {
    channelRef.current?.postMessage({ type: 'LOGOUT' } satisfies AuthMessage);
  }, []);

  return { broadcastLogout };
};
