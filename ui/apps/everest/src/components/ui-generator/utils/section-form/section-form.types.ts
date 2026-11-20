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

import { FormMode, Section, TopologyUISchemas } from '../../ui-generator.types';
import { FieldOverrides } from '../field-overrides';

// The schema the API checks one object of the form against, such as a
// class's parameters schema for the object at "parameters".
export interface SectionFormContract {
  // Where the object sits in the form's values; empty for the whole form.
  path: string;
  // OpenAPI v3.
  schema: unknown;
}

export interface SectionFormOptions {
  formMode?: FormMode;
  overrides?: FieldOverrides;
  contract?: SectionFormContract;
}

// A form rendered on its own rather than as a step of a provider topology,
// such as a class's credentials form, prepared by the same pipeline stages.
export interface SectionForm {
  sectionKey: string;
  formMode: FormMode;
  sections: Record<string, Section>;
  // The section wrapped as the one-topology schema the stages take.
  schema: TopologyUISchemas;
  contract?: SectionFormContract;
}
