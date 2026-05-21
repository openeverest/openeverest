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

import { useDefaultValues } from 'components/ui-generator/hooks/use-default-values';
import { TopologyUISchemas } from 'components/ui-generator/ui-generator.types';
import { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { WizardMode } from 'shared-types/wizard.types';
import { getDbWizardDefaultValues } from '../utils/get-default-values';

export const useDatabasePageDefaultValues = (
  mode: WizardMode,
  uiSchema: TopologyUISchemas,
  defaultSelectedTopology: string
): {
  // TODO add typescript types
  defaultValues: Record<string, unknown>;
  dbClusterData: Record<string, unknown>;
  dbClusterRequestStatus: 'error' | 'idle' | 'pending' | 'success';
  isFetching: boolean;
} => {
  const { state } = useLocation();

  const defaultSchemaValues = useDefaultValues(
    uiSchema,
    defaultSelectedTopology
  );

  const defaultValues = useMemo(() => {
    const providerName =
      state?.selectedDbProvider?.metadata?.name || 'unknown-provider';
    if (mode === WizardMode.New) {
      const dbWizardDefaultValues = getDbWizardDefaultValues(providerName);
      // Add topology to default values
      return {
        ...defaultSchemaValues,
        ...dbWizardDefaultValues,
        topology: { type: defaultSelectedTopology },
        backup: { schedules: [], classRef: { name: '' } },
      };
    } else {
      // TODO edit,restore,templates mode
      // When edit mode is implemented, map instance.spec.backup to form:
      //   backup.schedules = flattenSchedules(instance) → FlattenedSchedule[]
      //   backup.classRef.name = instance.spec.backup.classRef.name
      return {
        ...defaultSchemaValues,
        topology: { type: defaultSelectedTopology },
        backup: { schedules: [], classRef: { name: '' } },
      };
      //   return dbClusterRequestStatus === 'success'
      //     ? DbClusterPayloadToFormValues(dbCluster, mode, namespace)
      //     : defaults;
    }
  }, [
    defaultSchemaValues,
    defaultSelectedTopology,
    mode,
    state?.selectedDbEngine,
  ]);

  // TODO edit,restore,templates mode
  //   useEffect(() => {
  //     // dbClusterRequestStatus === 'success' when the request is enabled, which only happens if shouldRetrieveDbClusterData === true
  //     // hence, no need to re-check mode and so on here
  //     if (dbClusterRequestStatus === 'success' && dbCluster) {
  //       setDefaultValues(
  //         DbClusterPayloadToFormValues(dbCluster, mode, namespace)
  //       );
  //     }
  //   }, [dbCluster, dbClusterRequestStatus, mode, namespace]);

  return {
    defaultValues,
    dbClusterData: {},
    // TODO change when api is ready
    dbClusterRequestStatus: 'success',
    // TODO change when api is ready
    isFetching: false,
  };
};
