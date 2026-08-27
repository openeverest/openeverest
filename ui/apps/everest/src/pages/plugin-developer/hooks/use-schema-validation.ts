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

import { useMemo } from 'react';
import { TopologyUISchemas } from 'components/ui-generator/ui-generator.types';
import { validateSchema } from '../utils/validate-schema';
import { Diagnostic } from '../editor/types';

type SchemaValidation = {
  diagnostics: Diagnostic[];
  parsed: TopologyUISchemas | null;
};

// Derived, not state, so every path that changes the text stays in sync.
export const useSchemaValidation = (yamlText: string): SchemaValidation =>
  useMemo(() => validateSchema(yamlText), [yamlText]);
