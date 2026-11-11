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

import { Provider } from 'shared-types/api.types';
import { FormMode, Section } from '../../ui-generator.types';
import { preprocessSchema } from './preprocess-schema';

// The pipeline stages take a topology, so a lone section is wrapped as one.
export const preprocessSection = (
  sectionKey: string,
  section: Section,
  formMode: FormMode,
  providerObject?: Provider
): Record<string, Section> =>
  preprocessSchema(
    { [sectionKey]: { sections: { [sectionKey]: section } } },
    providerObject,
    formMode
  )[sectionKey].sections;
