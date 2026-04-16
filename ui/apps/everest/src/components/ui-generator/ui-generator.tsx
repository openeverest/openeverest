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

import React from 'react';
import { FormGroup, Stack, Typography } from '@mui/material';
import { Section, FormMode } from './ui-generator.types';
import { orderComponents, renderComponent } from './utils/component-renderer';
import { UiGeneratorProvider } from './ui-generator-context';
import { Provider } from 'shared-types/api.types';

type UIGeneratorProps = {
  /** Section key identifying which section to render. */
  sectionKey: string;
  sections: { [key: string]: Section };
  providerObject?: Provider;
  loadingDefaultsForEdition?: boolean;
  formMode?: FormMode;
};

export const UIGenerator = ({
  sectionKey,
  sections,
  providerObject,
  loadingDefaultsForEdition,
  formMode,
}: UIGeneratorProps) => {
  const section = sections[sectionKey];
  const components = section?.components;

  if (!components || Object.keys(components).length === 0) {
    return <Typography>No components available for this step</Typography>;
  }

  const orderedComponents = orderComponents(
    components,
    section?.componentsOrder
  );

  // Build base path for field names (no topology key since it's already selected)
  const basePath = sectionKey || '';

  return (
    <UiGeneratorProvider
      providerObject={providerObject}
      loadingDefaultsForEdition={loadingDefaultsForEdition}
      formMode={formMode}
    >
      <FormGroup sx={{ mt: 3 }}>
        <Stack spacing={2}>
          {orderedComponents.map(([key, item]) => {
            const fieldName = basePath ? `${basePath}.${key}` : key;
            return (
              <React.Fragment key={fieldName}>
                {renderComponent({
                  item,
                  name: fieldName,
                })}
              </React.Fragment>
            );
          })}
        </Stack>
      </FormGroup>
    </UiGeneratorProvider>
  );
};
