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

import { isRestoreDataSourceInput } from 'hooks/api/restores/restore-data-source';
import { RestoreDataSourceInput } from 'shared-types/restores.types';

export interface RestoreNavigationRouteState {
  selectedDbCluster: string;
  namespace: string;
  restoreDataSource: RestoreDataSourceInput;
}

export interface ParsedRestoreNavigationState {
  sourceInstanceName?: string;
  sourceNamespace?: string;
  restoreDataSource?: RestoreDataSourceInput;
  isRestore: boolean;
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const readString = (value: unknown, key: string): string | undefined => {
  const fieldValue = isRecord(value) ? Reflect.get(value, key) : undefined;
  return typeof fieldValue === 'string' ? fieldValue : undefined;
};

const readRestoreDataSource = (
  value: unknown
): RestoreDataSourceInput | undefined => {
  if (!isRecord(value)) {
    return undefined;
  }

  const dataSource = Reflect.get(value, 'restoreDataSource');
  return isRestoreDataSourceInput(dataSource) ? dataSource : undefined;
};

export const buildRestoreNavigationRouteState = (
  sourceInstanceName: string,
  sourceNamespace: string,
  restoreDataSource: RestoreDataSourceInput
): RestoreNavigationRouteState => ({
  selectedDbCluster: sourceInstanceName,
  namespace: sourceNamespace,
  restoreDataSource,
});

export const parseRestoreNavigationRouteState = (
  state: unknown
): ParsedRestoreNavigationState => {
  const sourceInstanceName = readString(state, 'selectedDbCluster');
  const sourceNamespace = readString(state, 'namespace');
  const restoreDataSource = readRestoreDataSource(state);

  return {
    sourceInstanceName,
    sourceNamespace,
    restoreDataSource,
    isRestore: !!sourceInstanceName && !!sourceNamespace && !!restoreDataSource,
  };
};
