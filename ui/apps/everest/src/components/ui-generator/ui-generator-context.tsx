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

import { createContext, useContext, ReactNode } from 'react';
import { Provider } from 'shared-types/api.types';
import type { FormMode } from './ui-generator.types';

type UiGeneratorContextValue = {
  providerObject?: Provider;
  loadingDefaultsForEdition?: boolean;
  formMode?: FormMode;
};

const UiGeneratorContext = createContext<UiGeneratorContextValue | null>(null);

type UiGeneratorProviderProps = {
  providerObject?: Provider;
  loadingDefaultsForEdition?: boolean;
  formMode?: FormMode;
  children: ReactNode;
};

export const UiGeneratorProvider = ({
  providerObject,
  loadingDefaultsForEdition,
  formMode,
  children,
}: UiGeneratorProviderProps) => {
  return (
    <UiGeneratorContext.Provider
      value={{
        providerObject,
        loadingDefaultsForEdition,
        formMode,
      }}
    >
      {children}
    </UiGeneratorContext.Provider>
  );
};

export const useUiGeneratorContext = () => {
  const context = useContext(UiGeneratorContext);
  // Context might be null if used outside provider, return empty object for safety
  return (
    context || {
      providerObject: undefined,
      loadingDefaultsForEdition: false,
      formMode: undefined,
    }
  );
};
