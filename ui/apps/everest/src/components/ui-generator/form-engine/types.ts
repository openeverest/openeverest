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

import { ComponentType } from 'react';
import { z } from 'zod';
import { FormMode, Section, TopologyUISchemas } from '../ui-generator.types';
import { Provider } from 'shared-types/api.types';

// Step - a single unit in the multi-step wizard.  Both "static" steps (hand-coded
// components like BaseInfoStep) and "generated" steps (coming from the
// ui-schema sections) conform to the same contract.

export type StepProps = {
  loadingDefaultsForEdition?: boolean;
};

export type StepDefinition = {
  /** Unique step identifier (e.g. 'base', 'import', 'section:resources'). */
  id: string;
  /**
   * Origin of the step. Defaults to 'static' for hand-coded steps and
   * 'generated' for steps produced from the topology's UI schema. Plugin-
   * contributed steps set 'plugin' so the wizard can route validity
   * gating to the per-plugin validity map.
   */
  kind?: 'static' | 'generated' | 'plugin';
  /** For plugin steps — the plugin name (matches the Plugin CR name). */
  pluginName?: string;
  /**
   * For plugin steps — if true the wizard's Next/Submit button is enabled
   * even when the plugin has not called `onValidityChange`.
   */
  optional?: boolean;
  /** Human-readable title shown in the wizard header / sidebar. */
  label: string;
  /** Optional longer description. */
  description?: string;
  /** For generated steps — the key inside the topology's `sections` map. */
  sectionKey?: string;
  /** React component rendered when this step is active. */
  component: ComponentType<StepProps>;
  /** Field paths that belong to this step (used for error → step routing). */
  fields: string[];
};

export type FormEngineConfig = {
  // Pre-processed topology UI schema
  uiSchema: TopologyUISchemas;
  selectedTopology: string;
  staticSteps?: StepDefinition[];
  /**
   * Steps appended after the schema-generated steps. Used by the host to
   * inject plugin-contributed wizard steps at the end of the flow.
   */
  appendSteps?: StepDefinition[];
  providerObject?: Provider;
  namespace?: string;
  formMode?: FormMode;
};

export type FormEngineResult = {
  /** Ordered list of all steps (static + generated). */
  steps: StepDefinition[];
  /** Topology sections extracted from the schema. */
  sections: Record<string, Section>;
  /** Explicit section ordering (if specified in the schema). */
  sectionsOrder?: string[];
  /** Combined Zod schema generated from the ui-schema. */
  zodSchema: z.ZodTypeAny;
  /** CEL dependency groups for cross-field validation. */
  celDependencyGroups: string[][];
  /** Map from field path → step ID (for error routing). */
  fieldToStepMap: Record<string, string>;
  /** Schema-derived default values (nested object). */
  defaultValues: Record<string, unknown>;
  /** Post-process submitted form data (multipath expansion, empty removal). */
  postprocess: (data: Record<string, unknown>) => Record<string, unknown>;
};
