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

import { createContext, useContext, useEffect, useState } from 'react';

const REFRESH_INTERVAL_MS = 30 * 1000;

// Pins the time useNow returns, for what has to render the same every time.
export const NowContext = createContext<Date | undefined>(undefined);

// The current time, refreshed often enough for "since 4 min" to stay true
// between two fetches.
export const useNow = () => {
  const pinned = useContext(NowContext);
  const [now, setNow] = useState(() => new Date());

  useEffect(() => {
    if (pinned) {
      return;
    }
    const interval = setInterval(() => setNow(new Date()), REFRESH_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [pinned]);

  return pinned ?? now;
};
