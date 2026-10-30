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
import { useDeleteMonitoringDestination } from 'hooks/api/monitoring-destinations';
import { getSecretDeletedWithDestination } from 'utils/monitoring-destinations';
import {
  MonitoringDestinationDeletion,
  MonitoringDestinationRow,
} from '../monitoring-destinations.types';

// Deleting asks the server who uses the destination first: one still in use
// is not deleted. A Secret the form created for it goes with it.
export const useDestinationDeletion = (rows: MonitoringDestinationRow[]) => {
  const [deletion, setDeletion] = useState<MonitoringDestinationDeletion>();
  const fetchBindings = useFetchMonitoringBindings();
  const { mutate: deleteDestination, isPending: isDeleting } =
    useDeleteMonitoringDestination();

  const namespaceDestinationsOf = (namespace: string) =>
    rows.filter((row) => row.namespace === namespace).map((row) => row.raw);

  const request = (row: MonitoringDestinationRow) =>
    fetchBindings(row.namespace, row.name)
      .then((bindings) =>
        setDeletion({
          row,
          usedBy: bindings.map((binding) => binding.spec.instanceRef.name),
          secretDeletedToo: getSecretDeletedWithDestination(
            row.raw,
            namespaceDestinationsOf(row.namespace)
          ),
        })
      )
      // The request already reported the error.
      .catch(() => undefined);

  const confirm = () => {
    if (!deletion) {
      return;
    }
    const { namespace, raw } = deletion.row;
    deleteDestination(
      {
        namespace,
        destination: raw,
        namespaceDestinations: namespaceDestinationsOf(namespace),
      },
      { onSuccess: () => setDeletion(undefined) }
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
