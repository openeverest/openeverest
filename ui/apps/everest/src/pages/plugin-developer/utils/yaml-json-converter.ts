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

import { parse, stringify } from 'yaml';
import { TopologyUISchemas } from 'components/ui-generator/ui-generator.types';

export const yamlToJson = (yamlText: string): TopologyUISchemas => {
  const parsed = parse(yamlText);

  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('Top level YAML value must be an object');
  }

  return parsed as TopologyUISchemas;
};

export const formatYamlText = (yamlText: string): string => {
  const parsed = yamlToJson(yamlText);
  return stringify(parsed);
};
