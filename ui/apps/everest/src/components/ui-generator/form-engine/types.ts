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
import { Section, TopologyUISchemas } from '../ui-generator.types';
import { Provider } from 'types/api';

// Step - a single unit in the multi-step wizard.  Both "static" steps (hand-coded
// components like BaseInfoStep) and "generated" steps (coming from the
// ui-schema sections) conform to the same contract.

export type StepProps = {
  loadingDefaultsForEdition?: boolean;
};

export type StepDefinition = {
  /** Unique step identifier (e.g. 'base', 'import', 'section:resources'). */
  id: string;
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
  /** Pre-processed topology UI schema. */
  uiSchema: TopologyUISchemas;
  /** Currently selected topology key. */
  selectedTopology: string;
  /** Hand-coded steps injected before the schema-generated steps. */
  staticSteps?: StepDefinition[];
  /** Provider object passed down to UIGenerator for option resolution. */
  providerObject?: Provider;
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
