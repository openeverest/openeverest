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

import { TopologyUISchemas } from 'components/ui-generator/ui-generator.types';
import { useState } from 'react';

export const useTopology = (
  schema: TopologyUISchemas
): {
  topologies: string[];
  hasMultipleTopologies: boolean;
  selectedTopology: string;
  setSelectedTopology: React.Dispatch<React.SetStateAction<string>>;
} => {
  const topologies = Object.keys(schema);
  const defaultTopology = topologies[0] || '';
  const hasMultipleTopologies = topologies.length > 1;
  const [selectedTopology, setSelectedTopology] =
    useState<string>(defaultTopology);

  // Fall back to the first topology when the selection goes stale after a
  // schema change (e.g. live-edited in the playground). No-op for a fixed schema.
  const effectiveTopology = topologies.includes(selectedTopology)
    ? selectedTopology
    : defaultTopology;

  return {
    topologies,
    hasMultipleTopologies,
    selectedTopology: effectiveTopology,
    setSelectedTopology,
  };
};
