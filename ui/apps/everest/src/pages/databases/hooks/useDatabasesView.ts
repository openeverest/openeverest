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
import { useNamespaces } from 'hooks/api/namespaces/useNamespaces';
import { useProviders } from 'hooks/api/providers/useProviders';
import { useInstancesForNamespaces } from 'hooks/api/db-instances/useDbInstanceList';
import { useNamespacePermissionsForResource } from 'hooks/rbac';
import { useExtensionCatalog } from 'hooks/api/extension-catalog';
import { convertDbInstancesPayloadToTableFormat } from '../DbClusterView.utils';
import { InstanceTableElement } from '../dbClusterView.types';

export type DatabasesViewState =
  | { status: 'loading' }
  | { status: 'no-namespaces' }
  | { status: 'empty' }
  | { status: 'table'; rows: InstanceTableElement[] };

export type DatabasesView = {
  state: DatabasesViewState;
  namespaces: string[];
  providersNamesFilter: { text: string; value: string }[];
  hasCreatePermission: boolean;
  canAddCluster: boolean;
};

// Coordinates the five async sources this page depends on (namespaces,
// instances, providers, plugin-hub catalog, RBAC) into a single readiness +
// decision. The rule: never surface a branch until the data that branch needs
// has settled, so the page transitions straight from a neutral skeleton to its
// final form instead of flashing partial/stale states.
export const useDatabasesView = (): DatabasesView => {
  const { data: namespaces = [], isLoading: loadingNamespaces } = useNamespaces(
    {
      refetchInterval: 10 * 1000,
    }
  );
  const { data: providers = [], isLoading: loadingProviders } = useProviders();
  const { canCreate, isPermissionsLoading } =
    useNamespacePermissionsForResource('database-clusters');
  const { isLoading: catalogLoading } = useExtensionCatalog();

  const instancesResults = useInstancesForNamespaces(
    namespaces.map((ns) => ({ namespace: ns }))
  );
  const instancesLoading = instancesResults.some(
    (result) => result.queryResult.isLoading
  );
  // True once every instances query has completed a fetch since this mount.
  // On remount the cache may hold a now-stale empty list (e.g. right after
  // creating a DB in the wizard); waiting for the mount refetch avoids briefly
  // showing the empty state before the freshly created row arrives.
  const instancesFetchedAfterMount = instancesResults.every(
    (result) => result.queryResult.isFetchedAfterMount
  );

  const rows = useMemo(
    () => convertDbInstancesPayloadToTableFormat(instancesResults),
    [instancesResults]
  );

  const hasCreatePermission = canCreate.length > 0;
  const canAddCluster = hasCreatePermission && providers.length > 0;

  const providersNamesFilter = providers.reduce<
    { text: string; value: string }[]
  >((acc, p) => {
    const name = p.metadata?.name;
    if (name) acc.push({ text: name, value: name });
    return acc;
  }, []);

  let state: DatabasesViewState;
  if (instancesLoading || loadingNamespaces) {
    state = { status: 'loading' };
  } else if (rows.length > 0) {
    state = { status: 'table', rows };
  } else if (namespaces.length === 0) {
    state = { status: 'no-namespaces' };
  } else if (
    // Empty databases branch: hold the neutral skeleton until everything the
    // empty state needs to pick its final form (providers list, plugin-hub
    // catalog, RBAC) has resolved and a fresh instances fetch has confirmed the
    // list is genuinely empty rather than a stale pre-create cache.
    !instancesFetchedAfterMount ||
    loadingProviders ||
    catalogLoading ||
    isPermissionsLoading
  ) {
    state = { status: 'loading' };
  } else {
    state = { status: 'empty' };
  }

  return {
    state,
    namespaces,
    providersNamesFilter,
    hasCreatePermission,
    canAddCluster,
  };
};
