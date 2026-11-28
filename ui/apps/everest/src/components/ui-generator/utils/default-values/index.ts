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

import {
  Topology,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { buildDefaultsFromComponents } from './build-defaults-from-components';
import { convertToNestedObject } from './convert-to-nested-object';

// Processes all sections and converts flat defaults to nested object structure.
export const getDefaultValues = (
  schema: TopologyUISchemas,
  selectedTopology: string
): Record<string, unknown> => {
  const topology: Topology = schema[selectedTopology];

  if (!topology || !topology.sections) {
    return {};
  }

  const flatDefaults: Record<string, unknown> = {};

  // Build defaults from all sections
  Object.entries(topology.sections).forEach(([sectionKey, section]) => {
    if (section?.components) {
      const sectionDefaults = buildDefaultsFromComponents(
        section.components,
        sectionKey ? [sectionKey] : []
      );
      Object.assign(flatDefaults, sectionDefaults);
    }
  });

  return convertToNestedObject(flatDefaults);
};
