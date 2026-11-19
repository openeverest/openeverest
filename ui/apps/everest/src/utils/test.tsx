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

/* eslint-disable react-refresh/only-export-components */
import React from 'react';
import mediaQuery from 'css-mediaquery';
import { MemoryRouter } from 'react-router-dom';
import { ThemeContextProvider, everestThemeOptions } from '@percona/design';
import { ClusterContext } from 'contexts/clusters/cluster';

const TEST_CLUSTER_NAME = 'main';

export const createMatchMedia = (width: number) => {
  return (query: string) => {
    return {
      matches: mediaQuery.match(query, { width }),
      media: '',
      addListener: () => {},
      removeListener: () => {},
      onchange: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => true,
    };
  };
};

export const resizeScreenSize = (width: number) => {
  window.matchMedia = createMatchMedia(width);
};

export const TestWrapper = ({ children }: { children: React.ReactNode }) => (
  <MemoryRouter>
    <ClusterContext.Provider value={TEST_CLUSTER_NAME}>
      <ThemeContextProvider themeOptions={everestThemeOptions}>
        {children}
      </ThemeContextProvider>
    </ClusterContext.Provider>
  </MemoryRouter>
);
