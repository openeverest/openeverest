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

import { useState } from 'react';
import { useFetchMonitoringBindings } from 'hooks/api/monitoring-bindings';
import {
  useCanDeleteMonitoringDestination,
  useDeleteMonitoringDestination,
} from 'hooks/api/monitoring-destinations';
import { useNamespacePermissionsForResource } from 'hooks/rbac';
import { isDestinationInUseError } from 'utils/monitoring-destinations';
import {
  DestinationUsage,
  MonitoringDestinationDeletion,
  MonitoringDestinationRow,
} from '../monitoring-destinations.types';
import { toDestinationUsage } from '../monitoring-destinations.utils';

// The server decides whether it can go.
export const useDestinationDeletion = () => {
  const [deletion, setDeletion] = useState<MonitoringDestinationDeletion>();
  const canDelete = useCanDeleteMonitoringDestination();
  const fetchBindings = useFetchMonitoringBindings();
  const { canRead: bindingNamespaces } = useNamespacePermissionsForResource(
    'monitoring-bindings'
  );
  const { mutate: deleteDestination, isPending: isDeleting } =
    useDeleteMonitoringDestination();

  // Names come from the bindings the user may read; the count from core.
  const getUsage = async ({
    namespace,
    name,
    raw,
  }: MonitoringDestinationRow): Promise<DestinationUsage> => {
    const bindings = bindingNamespaces.includes(namespace)
      ? await fetchBindings(namespace, name)
      : [];
    const usage = toDestinationUsage(
      raw,
      bindings.map((binding) => binding.spec.instanceRef.name)
    );
    // The server refused, so someone uses it even if the count lags.
    return { ...usage, count: Math.max(usage.count, 1) };
  };

  const request = async (row: MonitoringDestinationRow) => {
    try {
      setDeletion(
        (await canDelete(row.namespace, row.name))
          ? { row }
          : { row, inUse: await getUsage(row) }
      );
    } catch {
      // The request already reported the error.
    }
  };

  const confirm = () => {
    if (!deletion) {
      return;
    }
    const { row } = deletion;
    deleteDestination(
      { namespace: row.namespace, name: row.name },
      {
        onSuccess: () => setDeletion(undefined),
        // An Instance started using it after the dry run.
        onError: (error) => {
          if (isDestinationInUseError(error)) {
            request(row);
          }
        },
      }
    );
  };

  return {
    deletion,
    isDeleting,
    request,
    confirm,
    cancel: () => setDeletion(undefined),
  };
};
