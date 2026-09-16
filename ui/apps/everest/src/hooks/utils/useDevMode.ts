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

import { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';

export const DEV_MODE_STORAGE_KEY = 'everest.devMode';

const readStoredDevMode = (): boolean => {
  try {
    return localStorage.getItem(DEV_MODE_STORAGE_KEY) === 'true';
  } catch {
    return false;
  }
};

export const useDevMode = (): boolean => {
  const { search } = useLocation();
  const [devMode, setDevMode] = useState<boolean>(readStoredDevMode);

  useEffect(() => {
    const param = new URLSearchParams(search).get('dev');
    if (param !== 'true' && param !== 'false') {
      return;
    }

    const next = param === 'true';
    setDevMode(next);
    try {
      localStorage.setItem(DEV_MODE_STORAGE_KEY, String(next));
    } catch {
      // localStorage unavailable (e.g. private mode); dev mode just won't persist.
    }
  }, [search]);

  return devMode;
};
