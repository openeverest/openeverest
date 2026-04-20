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
import { preprocessSchema } from 'components/ui-generator/utils/preprocess/preprocess-schema';
import { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { Provider } from 'shared-types/api.types';

export const useSchema = (): {
  uiSchema: TopologyUISchemas;
  topologies: string[];
  hasMultipleTopologies: boolean;
} => {
  const { state } = useLocation();
  const selectedDbProvider = state?.selectedDbProvider as Provider;

  const uiSchema = useMemo(
    () =>
      preprocessSchema(
        selectedDbProvider?.spec?.uiSchema || {},
        selectedDbProvider
      ),
    [selectedDbProvider]
  );

  const topologies = useMemo(
    () => (uiSchema ? Object.keys(uiSchema) : []),
    [uiSchema]
  );

  const hasMultipleTopologies = useMemo(
    () => topologies.length > 1,
    [topologies.length]
  );

  return { uiSchema, topologies, hasMultipleTopologies };
};
