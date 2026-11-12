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

import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { NowContext } from 'hooks/utils/useNow';
import {
  getFixtureInstance,
  getInstanceStatusFixture,
} from 'utils/tests/instance-status-fixtures';
import { DbClusterDetails } from './db-cluster-details';
import { DbInstanceContext } from './dbCluster.context';
import { REASON_HINTS } from './status-alert/status-alert.messages';

vi.mock('components/db-actions/db-actions', () => ({
  default: () => <div data-testid="db-actions" />,
}));

vi.mock('contexts/plugins', () => ({
  usePlugins: () => ({ plugins: [] }),
}));

// Renders the details of a database in the state a status fixture describes,
// at the time the fixture was taken.
const renderDetails = (fixture: string) =>
  render(
    <MemoryRouter initialEntries={['/databases/demo/db/overview']}>
      <NowContext.Provider
        value={new Date(getInstanceStatusFixture(fixture).now)}
      >
        <Routes>
          <Route
            path="/databases/:namespace/:instanceName/:tabs"
            element={
              <DbInstanceContext.Provider
                value={{
                  instance: getFixtureInstance(fixture),
                  isLoading: false,
                  instanceDeleted: false,
                  canReadCredentials: true,
                  isCredentialsPermissionLoading: false,
                }}
              >
                <DbClusterDetails />
              </DbInstanceContext.Provider>
            }
          >
            <Route path="overview" element={<div>Overview content</div>} />
          </Route>
        </Routes>
      </NowContext.Provider>
    </MemoryRouter>
  );

describe('DbClusterDetails', () => {
  it.each([
    ['healthy', 'Up · Ready'],
    ['rolling-upgrade', 'Up · Upgrading'],
    ['crash-loop-all-replicas-stalled', 'Down · Crash loop'],
    // A database that has never been up is not down.
    ['creating', 'Creating'],
    // Nor is one whose availability nobody can tell.
    ['change-availability-unknown', 'Applying changes'],
    ['deleting-waiting-for-children', 'Deleting · Waiting for backups'],
  ])(
    'says whether the %s database is up and what is happening in one line',
    (fixture, sentence) => {
      renderDetails(fixture);

      expect(screen.getByTestId('db-status')).toHaveTextContent(sentence);
    }
  );

  it('explains the status and says since when', () => {
    renderDetails('rolling-upgrade');

    expect(screen.getByTestId('instance-status-message')).toHaveTextContent(
      'Rolling out version 8.0.12 (1 of 3 pods updated)'
    );
    expect(screen.getByTestId('instance-status-since')).toHaveTextContent(
      'since 4 min'
    );
  });

  it('reads as being created before any provider has reported', () => {
    renderDetails('never-reconciled-fresh');

    expect(screen.getByTestId('db-status')).toHaveTextContent('Creating');
    expect(
      screen.queryByTestId('instance-status-since')
    ).not.toBeInTheDocument();
  });

  it.each(['healthy', 'rolling-upgrade', 'creating', 'stopped', 'deleting'])(
    'raises no alert for the %s database',
    (fixture) => {
      renderDetails(fixture);

      expect(
        screen.queryByTestId('instance-status-alert')
      ).not.toBeInTheDocument();
    }
  );

  it('warns about a problem that may still clear, quoting it', () => {
    renderDetails('crash-loop-one-replica');

    const alert = screen.getByTestId('instance-status-alert');
    expect(alert).toHaveClass('MuiAlert-colorWarning');
    expect(alert).toHaveTextContent('Crash loop');
    expect(alert).toHaveTextContent(REASON_HINTS.CrashLoopBackOff);
    expect(alert).toHaveTextContent(
      'engine: container mongod last exited with Error (exit code 14)'
    );
    expect(alert).toHaveTextContent('First observed 5 min ago.');
  });

  it('reports a problem that needs someone as an error', () => {
    renderDetails('secret-missing-stalled');

    const alert = screen.getByTestId('instance-status-alert');
    expect(alert).toHaveClass('MuiAlert-colorError');
    expect(alert).toHaveTextContent('Secret not found');
    expect(alert).toHaveTextContent(
      'secret app-users named by spec.userSecretRef not found in demo'
    );
  });

  it('alerts about a problem it has no advice for', () => {
    renderDetails('progress-deadline-exceeded');

    const alert = screen.getByTestId('instance-status-alert');
    expect(alert).toHaveTextContent('Not progressing');
    expect(alert).toHaveTextContent(
      'No progress for 30m0s: waiting for the replica set to initialise'
    );
  });

  it('does not alert about a problem of the spec that was just replaced', () => {
    renderDetails('fix-pushed-to-stalled');

    expect(screen.getByTestId('db-status')).toHaveTextContent('Creating');
    expect(
      screen.queryByTestId('instance-status-alert')
    ).not.toBeInTheDocument();
  });
});
