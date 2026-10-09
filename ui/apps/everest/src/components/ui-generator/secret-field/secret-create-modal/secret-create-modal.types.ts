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
import { Provider } from 'shared-types/api.types';
import {
  Section,
  UIGeneratorProps,
} from 'components/ui-generator/ui-generator.types';
import { SECRET_DATA_ROOT } from './secret-create-modal.constants';

export interface SecretCreateModalProps {
  definition: string;
  // The definition's create form (see getSecretDefinitionSection).
  section: Section;
  providerObject: Provider;
  namespace: string;
  // The enclosing UIGenerator (UiGeneratorContext.generator).
  generator: ComponentType<UIGeneratorProps>;
  onClose: () => void;
  onCreated: (name: string) => void;
}

export type SecretFormData = { metadata: { name: string } } & Record<
  typeof SECRET_DATA_ROOT,
  Record<string, unknown>
>;
