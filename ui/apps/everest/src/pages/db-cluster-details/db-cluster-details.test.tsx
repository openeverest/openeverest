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

import { render, screen, within } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { DbClusterDetails } from './db-cluster-details';
import { DbInstanceContext } from './dbCluster.context';
import type { DbInstanceContextProps } from './dbCluster.context.types';
import {
  Messages as StatusAlertMessages,
  REASON_HINTS,
} from './status-alert/status-alert.messages';
import type { Instance } from 'shared-types/api.types';
import type { InstanceCondition } from 'shared-types/instance.types';

vi.mock('components/db-actions/db-actions', () => ({
  default: () => <div data-testid="db-actions" />,
}));

vi.mock('contexts/plugins', () => ({
  usePlugins: () => ({ plugins: [] }),
}));

const condition = (
  type: string,
  status: InstanceCondition['status'],
  reason = type,
  message = ''
): InstanceCondition => ({
  type,
  status,
  reason,
  message,
  lastTransitionTime: '2026-10-02T12:00:00Z',
});

const instanceWith = (
  conditions: InstanceCondition[],
  phase: NonNullable<Instance['status']>['phase'] = 'Provisioned',
  generation = 1
): Instance => {
  // The API returns the generation; the generated metadata type leaves it out.
  const metadata = { name: 'my-test-db', generation };

  return {
    apiVersion: 'core.openeverest.io/v1alpha1',
    kind: 'Instance',
    metadata,
    spec: {
      providerRef: { name: 'test-provider' },
      topology: { type: 'ha' },
    },
    status: { phase, observedGeneration: 1, conditions },
  };
};

const CRASH_LOOP_MESSAGE =
  'engine: 1 of 3 pods keep crashing: container mongod last exited with OOMKilled (exit code 137)';

const crashLoop = (stalled: boolean) => [
  condition('Available', 'False', 'NoPrimary'),
  condition('Ready', 'False', 'CrashLoopBackOff', CRASH_LOOP_MESSAGE),
  condition('Stalled', stalled ? 'True' : 'False', 'CrashLoopBackOff'),
  condition('NeedsAttention', 'True', 'CrashLoopBackOff', CRASH_LOOP_MESSAGE),
];

const renderDetails = (instance: Instance) => {
  const value: DbInstanceContextProps = {
    instance,
    isLoading: false,
    instanceDeleted: false,
    canReadCredentials: true,
    isCredentialsPermissionLoading: false,
  };

  return render(
    <MemoryRouter initialEntries={['/databases/test-ns/my-test-db/overview']}>
      <Routes>
        <Route
          path="/databases/:namespace/:instanceName/:tabs"
          element={
            <DbInstanceContext.Provider value={value}>
              <DbClusterDetails />
            </DbInstanceContext.Provider>
          }
        >
          <Route path="overview" element={<div>Overview content</div>} />
        </Route>
      </Routes>
    </MemoryRouter>
  );
};

describe('DbClusterDetails', () => {
  it('says in the header that an Instance being upgraded is still available', () => {
    renderDetails(
      instanceWith([
        condition('Available', 'True'),
        condition('Ready', 'False', 'Upgrading'),
        condition('Upgrading', 'True', 'RollingOutImage'),
      ])
    );

    expect(screen.getByTestId('my-test-db-status')).toHaveTextContent(
      'Upgrading \u00b7 available'
    );
    expect(
      screen.queryByTestId('instance-status-alert')
    ).not.toBeInTheDocument();
  });

  it('does not call an Instance that has never been up unavailable', () => {
    renderDetails(
      instanceWith(
        [
          condition('Available', 'False', 'NotCreated'),
          condition('Ready', 'False', 'Creating'),
        ],
        'Pending'
      )
    );

    expect(screen.getByTestId('my-test-db-status')).toHaveTextContent(
      /^Creating$/
    );
  });

  it('warns about a problem that may still clear, with advice and the reported cause', () => {
    renderDetails(instanceWith(crashLoop(false), 'Pending'));

    const alert = within(screen.getByTestId('instance-status-alert'));
    expect(alert.getByText('Crash loop')).toBeInTheDocument();
    expect(alert.getByText(REASON_HINTS.CrashLoopBackOff)).toBeInTheDocument();
    expect(
      alert.getByText(StatusAlertMessages.outlook.retrying)
    ).toBeInTheDocument();
    expect(alert.getByText(CRASH_LOOP_MESSAGE)).toBeInTheDocument();
  });

  it('says that waiting no longer helps once the Instance is stalled', () => {
    renderDetails(instanceWith(crashLoop(true), 'Pending'));

    const alert = within(screen.getByTestId('instance-status-alert'));
    expect(
      alert.getByText(StatusAlertMessages.outlook.stalled)
    ).toBeInTheDocument();
    expect(
      alert.queryByText(StatusAlertMessages.outlook.retrying)
    ).not.toBeInTheDocument();
  });

  it('shows a problem the UI has no advice for as the provider reported it', () => {
    const message = 'spec.pause is set on the PerconaServerMongoDB object';
    renderDetails(
      instanceWith([
        condition('Ready', 'False', 'PausedOutsideOpenEverest', message),
        condition('Stalled', 'True', 'PausedOutsideOpenEverest'),
        condition(
          'NeedsAttention',
          'True',
          'PausedOutsideOpenEverest',
          message
        ),
      ])
    );

    const alert = within(screen.getByTestId('instance-status-alert'));
    expect(alert.getByText('Paused outside open everest')).toBeInTheDocument();
    expect(alert.getByText(message)).toBeInTheDocument();
  });

  it('promises no retries for an Instance whose provider is not installed', () => {
    const message = 'Provider "percona-server-mongodb" is not installed.';
    renderDetails(
      instanceWith(
        [condition('Unattended', 'True', 'ProviderNotFound', message)],
        'Pending',
        2
      )
    );

    const alert = within(screen.getByTestId('instance-status-alert'));
    expect(alert.getByText('Provider not found')).toBeInTheDocument();
    expect(alert.getByText(REASON_HINTS.ProviderNotFound)).toBeInTheDocument();
    expect(alert.getByText(message)).toBeInTheDocument();
    expect(
      alert.queryByText(StatusAlertMessages.outlook.retrying)
    ).not.toBeInTheDocument();
  });

  it('drops the alert about an older spec once a fix is pushed', () => {
    renderDetails(instanceWith(crashLoop(true), 'Pending', 2));

    expect(
      screen.queryByTestId('instance-status-alert')
    ).not.toBeInTheDocument();
    expect(screen.getByTestId('my-test-db-status')).toHaveTextContent(
      'Creating'
    );
  });

  it('does not warn about pods that are only starting', () => {
    renderDetails(
      instanceWith(
        [
          condition('Ready', 'False', 'Creating'),
          condition('NeedsAttention', 'False', 'NoProblems'),
          condition('PodsReady', 'Unknown', 'NotReady'),
        ],
        'Pending'
      )
    );

    expect(
      screen.queryByTestId('instance-status-alert')
    ).not.toBeInTheDocument();
  });
});
