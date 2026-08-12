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

import { createContext, useContext } from 'react';
import {
  Section,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { Provider } from 'shared-types/api.types';
import type { PluginStepSummary } from './plugin-form-steps';

type DatabaseFormContextType = {
  uiSchema: TopologyUISchemas;
  topologies: string[];
  hasMultipleTopologies: boolean;
  defaultTopology: string;
  sections: { [key: string]: Section };
  sectionsOrder?: string[];
  providerObject?: Provider;
  hasBackupStep: boolean;
  /**
   * Per-plugin descriptors for the right-sidebar summary preview, in
   * wizard-step order. Empty when no plugin step extensions are active for
   * the current namespace + engine type.
   */
  pluginSummaries?: PluginStepSummary[];
  /**
   * Returns the latest config blob the named plugin step pushed via
   * `onChange`, or `undefined` if the user hasn't visited that step yet.
   * Returns `undefined` outside of contexts that wire plugin steps.
   */
  getPluginConfig?: (pluginName: string) => Record<string, unknown> | undefined;
};

const DatabaseFormContext = createContext<DatabaseFormContextType | null>(null);

export const DatabaseFormProvider = DatabaseFormContext.Provider;

export const useDatabaseFormContext = () => {
  const context = useContext(DatabaseFormContext);
  if (!context) {
    throw new Error(
      'useDatabaseFormContext must be used within DatabaseFormProvider'
    );
  }
  return context;
};
