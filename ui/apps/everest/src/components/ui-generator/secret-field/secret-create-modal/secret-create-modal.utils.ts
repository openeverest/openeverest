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

import { Provider, Secret } from 'shared-types/api.types';
import { generateShortUID } from 'utils/generateShortUID';
import {
  FormMode,
  Section,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { preprocessSchema } from 'components/ui-generator/utils/preprocess/preprocess-schema';
import { applyModeOverrides } from 'components/ui-generator/utils/preprocess/apply-mode-overrides';
import {
  MAX_SECRET_NAME_LENGTH,
  SECRET_DEFINITION_LABEL,
  SECRET_PROVIDER_LABEL,
  SECRET_SECTION_KEY,
} from './secret-create-modal.constants';

// `<definition>-<uid>`, cut to the name limit, with characters a name can't hold replaced.
export const generateSecretName = (definition: string): string => {
  const suffix = generateShortUID();
  const prefix = definition
    .toLowerCase()
    .replace(/[^a-z0-9-]/g, '-')
    .slice(0, MAX_SECRET_NAME_LENGTH - suffix.length - 1);
  return `${prefix}-${suffix}`;
};

// The pipeline stages take a topology, so the definition's one section is wrapped as one.
export const toSecretSchema = (
  section: Section,
  providerObject: Provider
): TopologyUISchemas => {
  const { sections } = preprocessSchema(
    { [SECRET_SECTION_KEY]: { sections: { [SECRET_SECTION_KEY]: section } } },
    providerObject
  )[SECRET_SECTION_KEY];
  return {
    [SECRET_SECTION_KEY]: {
      sections: applyModeOverrides(sections, FormMode.New),
    },
  };
};

export const buildSecretPayload = ({
  name,
  definition,
  provider,
  values,
}: {
  name: string;
  definition: string;
  provider: string;
  values: Record<string, unknown>;
}): Secret => ({
  metadata: {
    name,
    labels: {
      [SECRET_PROVIDER_LABEL]: provider,
      [SECRET_DEFINITION_LABEL]: definition,
    },
  },
  // Secret data is text; number and toggle fields are sent as their JSON form.
  stringData: Object.fromEntries(
    Object.entries(values).map(([key, value]) => [
      key,
      typeof value === 'string' ? value : JSON.stringify(value),
    ])
  ),
});
