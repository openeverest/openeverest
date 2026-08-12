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
  buildRestoreNavigationRouteState,
  parseRestoreNavigationRouteState,
} from './restore-navigation-state';

describe('restore-navigation-state', () => {
  it('builds and parses a valid backup restore route state', () => {
    const state = buildRestoreNavigationRouteState('source-db', 'ns-a', {
      type: 'Backup',
      backupName: 'daily-1',
      storageName: 's3-main',
    });

    expect(state).toEqual({
      selectedDbCluster: 'source-db',
      namespace: 'ns-a',
      restoreDataSource: {
        type: 'Backup',
        backupName: 'daily-1',
        storageName: 's3-main',
      },
    });

    expect(parseRestoreNavigationRouteState(state)).toEqual({
      sourceInstanceName: 'source-db',
      sourceNamespace: 'ns-a',
      restoreDataSource: {
        type: 'Backup',
        backupName: 'daily-1',
        storageName: 's3-main',
      },
      isRestore: true,
    });
  });

  it('keeps source identity but disables restore mode for invalid payload', () => {
    const parsed = parseRestoreNavigationRouteState({
      selectedDbCluster: 'source-db',
      namespace: 'ns-a',
      restoreDataSource: {
        type: 'PointInTime',
        recoveryTarget: 'date',
      },
    });

    expect(parsed).toEqual({
      sourceInstanceName: 'source-db',
      sourceNamespace: 'ns-a',
      restoreDataSource: undefined,
      isRestore: false,
    });
  });
});
