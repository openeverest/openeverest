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

import { fireEvent, render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { DbInstanceForNamespaceResult } from 'hooks/api/db-instances';
import { Instance } from 'shared-types/api.types';
import { getFixtureInstance } from 'utils/tests/instance-status-fixtures';
import { DbClusterView } from './DbClusterView';
import { convertDbInstancesPayloadToTableFormat } from './DbClusterView.utils';
import { useDatabasesView } from './hooks/useDatabasesView';

vi.mock('./hooks/useDatabasesView', () => ({ useDatabasesView: vi.fn() }));

vi.mock('components/db-actions/db-actions', () => ({
  DbActions: () => null,
}));

const NOW = new Date('2026-03-01T12:00:00Z');

// One database per fixture, named after it.
const database = (fixture: string, status?: Instance['status']): Instance => {
  const instance = getFixtureInstance(fixture);
  return {
    ...instance,
    metadata: { ...instance.metadata, name: fixture },
    status: status ?? instance.status,
  };
};

const renderList = (instances: Instance[]) => {
  vi.mocked(useDatabasesView).mockReturnValue({
    state: {
      status: 'table',
      rows: convertDbInstancesPayloadToTableFormat(
        [
          {
            namespace: 'demo',
            queryResult: { data: instances, isSuccess: true },
          } as DbInstanceForNamespaceResult,
        ],
        NOW
      ),
    },
    namespaces: ['demo'],
    providersNamesFilter: [],
    hasCreatePermission: false,
    canAddCluster: false,
  });

  return render(
    <MemoryRouter>
      <DbClusterView />
    </MemoryRouter>
  );
};

// The names of the rows, top to bottom.
const listedNames = () =>
  screen
    .getAllByTestId(/-available$/)
    .map((cell) =>
      cell.getAttribute('data-testid')!.replace(/-available$/, '')
    );

describe('DbClusterView status', () => {
  it('shows what is happening and whether the database is up in two cells', () => {
    renderList([
      database('rolling-upgrade'),
      database('in-place-restore'),
      database('change-availability-unknown'),
    ]);

    expect(screen.getByTestId('rolling-upgrade-status')).toHaveTextContent(
      'Upgrading'
    );
    expect(screen.getByTestId('rolling-upgrade-available')).toHaveTextContent(
      'Yes'
    );
    expect(screen.getByTestId('in-place-restore-status')).toHaveTextContent(
      'Restoring'
    );
    expect(screen.getByTestId('in-place-restore-available')).toHaveTextContent(
      'No'
    );
    expect(
      screen.getByTestId('change-availability-unknown-available')
    ).toHaveTextContent('\u2014');
  });

  it('lists what needs someone first, then the databases that are down', () => {
    renderList([
      database('healthy'),
      database('deleting'),
      database('stopped'),
      database('rolling-upgrade'),
      database('change-availability-unknown'),
      database('in-place-restore'),
      database('never-reconciled-old'),
      database('crash-loop-one-replica'),
      database('backup-storage-missing-stalled'),
      database('crash-loop-all-replicas-stalled'),
    ]);

    expect(listedNames()).toEqual([
      'crash-loop-all-replicas-stalled',
      'backup-storage-missing-stalled',
      'crash-loop-one-replica',
      'never-reconciled-old',
      'in-place-restore',
      'change-availability-unknown',
      'rolling-upgrade',
      'stopped',
      'deleting',
      'healthy',
    ]);
  });

  it('narrows the list to warnings and errors in one click, and back', () => {
    renderList([
      database('healthy'),
      database('rolling-upgrade'),
      database('crash-loop-one-replica'),
      database('rejected-spec'),
    ]);
    const shortcut = screen.getByTestId('needs-attention-filter');
    expect(shortcut).toHaveTextContent('Needs attention (2)');

    fireEvent.click(shortcut);

    expect(listedNames()).toEqual(['rejected-spec', 'crash-loop-one-replica']);

    fireEvent.click(shortcut);

    expect(listedNames()).toHaveLength(4);
  });

  it('has nothing to narrow to when no database has a problem', () => {
    renderList([database('healthy'), database('rolling-upgrade')]);

    expect(screen.getByTestId('needs-attention-filter')).toHaveAttribute(
      'aria-disabled',
      'true'
    );
  });

  it('marks a notice beside a status it leaves alone', () => {
    const healthy = database('healthy');
    renderList([
      database('healthy', {
        ...healthy.status,
        conditions: [
          ...(healthy.status?.conditions ?? []),
          {
            type: 'MaintenancePending',
            status: 'True',
            reason: 'AwaitingApproval',
            message: '1 action awaits approval',
            lastTransitionTime: '2026-03-01T11:00:00Z',
          },
          {
            type: 'MonitoringReady',
            status: 'False',
            reason: 'MonitoringNotReady',
            message: 'pmm: destination is not ready',
            lastTransitionTime: '2026-03-01T11:00:00Z',
          },
        ],
      }),
      database('rolling-upgrade'),
    ]);

    expect(screen.getByTestId('healthy-status')).toHaveTextContent('Ready');
    expect(screen.getByTestId('healthy-notices')).toHaveTextContent(
      '2 notices'
    );
    const upgrading = screen
      .getByTestId('rolling-upgrade-status')
      .closest('tr')!;
    expect(
      within(upgrading).queryByTestId('rolling-upgrade-notices')
    ).not.toBeInTheDocument();
  });
});
