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
import { toSecretStringData } from 'utils/secrets';
import {
  FormMode,
  Section,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { preprocessSection } from 'components/ui-generator/utils/preprocess/preprocess-section';
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

export const toSecretSchema = (
  section: Section,
  providerObject: Provider,
  formMode: FormMode
): TopologyUISchemas => ({
  [SECRET_SECTION_KEY]: {
    sections: preprocessSection(
      SECRET_SECTION_KEY,
      section,
      formMode,
      providerObject
    ),
  },
});

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
  stringData: toSecretStringData(values),
});
