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
import { extractInstanceValues } from 'components/ui-generator/utils/default-values/extract-instance-values';
import {
  FormMode,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { useDbInstance } from 'hooks/api/db-instances/useDbInstance';
import { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { getDbWizardDefaultValues } from '../utils/get-default-values';
import { generateShortUID } from 'utils/generateShortUID';

export const useDatabasePageDefaultValues = (
  mode: FormMode,
  uiSchema: TopologyUISchemas,
  defaultSelectedTopology: string,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _hasBackupStep = false
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

  // ── Restore mode: fetch source instance ──────────────────────────────────
  const isRestore = mode === FormMode.Restore;
  const sourceInstanceName = state?.selectedDbCluster as string | undefined;
  const sourceNamespace = state?.namespace as string | undefined;

  const {
    data: sourceInstance,
    status: sourceInstanceStatus,
    isFetching: sourceInstanceFetching,
  } = useDbInstance(sourceNamespace ?? '', sourceInstanceName ?? '', {
    enabled: isRestore && !!sourceInstanceName && !!sourceNamespace,
  });

  const defaultValues = useMemo(() => {
    const providerName =
      state?.selectedDbProvider?.metadata?.name || 'unknown-provider';

    if (mode === FormMode.New) {
      const dbWizardDefaultValues = getDbWizardDefaultValues(providerName);
      return {
        ...defaultSchemaValues,
        ...dbWizardDefaultValues,
        topology: { type: defaultSelectedTopology },
        backup: { schedules: [], classRef: { name: '' } },
      };
    }

    if (mode === FormMode.Restore) {
      if (!sourceInstance) {
        // Still loading — return schema defaults as placeholder
        return {
          ...defaultSchemaValues,
          topology: { type: defaultSelectedTopology },
          backup: { schedules: [], classRef: { name: '' } },
        };
      }

      const topologyType =
        (sourceInstance.spec?.topology?.type as string) ||
        defaultSelectedTopology;
      const sections = uiSchema[topologyType]?.sections ?? {};

      const extractedValues = extractInstanceValues(
        sections,
        sourceInstance as unknown as Record<string, unknown>,
        FormMode.Restore
      );

      return {
        ...defaultSchemaValues,
        ...extractedValues,
        topology: { type: topologyType },
        // Generate a new name for the restored instance
        dbName: `inst-${generateShortUID()}`,
        // Provider name from source instance
        provider: (sourceInstance.spec?.provider as string) ?? '',
        // Keep namespace from source
        k8sNamespace: sourceNamespace ?? '',
        // No backup schedules on new-from-restore
        backup: { schedules: [], classRef: { name: '' } },
      };
    }

    // Fallback (edit/import modes — not yet implemented)
    return {
      ...defaultSchemaValues,
      topology: { type: defaultSelectedTopology },
      backup: { schedules: [], classRef: { name: '' } },
    };
  }, [
    defaultSchemaValues,
    defaultSelectedTopology,
    mode,
    sourceInstance,
    sourceNamespace,
    state?.selectedDbProvider?.metadata?.name,
    uiSchema,
  ]);

  return {
    defaultValues,
    dbClusterData: sourceInstance
      ? (sourceInstance as unknown as Record<string, unknown>)
      : {},
    dbClusterRequestStatus: isRestore ? sourceInstanceStatus : 'success',
    isFetching: isRestore ? sourceInstanceFetching : false,
  };
};
