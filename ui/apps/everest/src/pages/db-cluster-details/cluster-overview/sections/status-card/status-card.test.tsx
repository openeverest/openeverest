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
import { NowContext } from 'hooks/utils/useNow';
import { Instance } from 'shared-types/api.types';
import { TestWrapper } from 'utils/test';
import {
  getFixtureInstance,
  getInstanceStatusFixture,
  withStatus,
} from 'utils/tests/instance-status-fixtures';
import { StatusCard } from './status-card';

// Renders the card at the time the fixture was taken.
const renderCard = (fixture: string, instance = getFixtureInstance(fixture)) =>
  render(
    <TestWrapper>
      <NowContext.Provider
        value={new Date(getInstanceStatusFixture(fixture).now)}
      >
        <StatusCard instance={instance} />
      </NowContext.Provider>
    </TestWrapper>
  );

const section = (name: string) => within(screen.getByTestId(`status-${name}`));

// A rolling upgrade as a provider reports it in full.
const upgrading = (): Instance =>
  withStatus(
    getFixtureInstance('rolling-upgrade'),
    [
      { type: 'ReferencesResolved', status: 'True', reason: 'Resolved' },
      { type: 'Synced', status: 'True', reason: 'Synced' },
      { type: 'PodsScheduled', status: 'True', reason: 'Scheduled' },
      { type: 'EngineHealthy', status: 'True', reason: 'Healthy' },
      {
        type: 'StorageResizing',
        status: 'False',
        reason: 'Succeeded',
        message: 'Volumes expanded to 50Gi',
        lastTransitionTime: '2026-03-01T09:00:00Z',
      },
      {
        type: 'MaintenancePending',
        status: 'True',
        reason: 'AwaitingApproval',
        message: '1 action awaits approval',
      },
      {
        type: 'example.com/ShardsBalanced',
        status: 'False',
        reason: 'Rebalancing',
        message: 'Moving chunks',
      },
    ],
    [
      { name: 'engine', replicas: 3, readyReplicas: 2 },
      { name: 'proxy', replicas: 2, readyReplicas: 2 },
    ]
  );

describe('StatusCard', () => {
  it('sums the status up in one line, with its explanation and its age', () => {
    renderCard('rolling-upgrade');

    const summary = section('summary');
    expect(summary.getByTestId('status-card-status')).toHaveTextContent(
      'Up \u00b7 Upgrading'
    );
    expect(summary.getByTestId('status-card-message')).toHaveTextContent(
      'Rolling out version 8.0.12 (1 of 3 pods updated)'
    );
    expect(summary.getByText('since 4 min')).toBeInTheDocument();
  });

  it('counts the ready pods of each component', () => {
    renderCard('rolling-upgrade', upgrading());

    expect(screen.getByTestId('status-component-engine')).toHaveTextContent(
      'engine 2/3 ready'
    );
    expect(screen.getByTestId('status-component-proxy')).toHaveTextContent(
      'proxy 2/2 ready'
    );
  });

  it('has no problems to list for a database that is only busy', () => {
    renderCard('rolling-upgrade', upgrading());

    expect(section('problems').getByText('None')).toBeInTheDocument();
  });

  it('lists a failing check as a problem, with its reason and message', () => {
    renderCard('crash-loop-one-replica');

    const problem = screen.getByTestId('status-problem-PodsReady');
    expect(problem).toHaveTextContent('Pods ready');
    expect(problem).toHaveTextContent('Crash loop');
    expect(problem).toHaveTextContent(
      'engine: container mongod last exited with Error (exit code 14)'
    );
  });

  it('lists a problem that is not a failing check', () => {
    renderCard('rejected-spec');

    const problem = screen.getByTestId('status-problem-NeedsAttention');
    expect(problem).toHaveTextContent('Invalid configuration');
    expect(problem).toHaveTextContent(
      'spec.components.engine.replicas: must be odd'
    );
  });

  it('folds the passing checks into a count that opens', () => {
    renderCard('rolling-upgrade', upgrading());
    const checks = section('checks');

    expect(checks.getByText('4 checks passing')).toBeInTheDocument();
    expect(checks.queryByText('Provider in sync')).not.toBeInTheDocument();

    fireEvent.click(checks.getByTestId('status-checks-toggle'));

    expect(checks.getByText('References resolved')).toBeInTheDocument();
    expect(checks.getByText('Provider in sync')).toBeInTheDocument();
    expect(checks.getByText('Pods scheduled')).toBeInTheDocument();
    expect(checks.getByText('Engine healthy')).toBeInTheDocument();
  });

  it('lists a check in progress under its plain title, with what it waits for', () => {
    renderCard('rolling-upgrade', upgrading());

    const check = screen.getByTestId('status-check-PodsReady');
    expect(check).toHaveTextContent('Pods ready');
    expect(check).toHaveTextContent('engine: 1 pod is not ready');
  });

  it('does not show a check the Instance does not report', () => {
    renderCard('rolling-upgrade', upgrading());
    const checks = section('checks');

    fireEvent.click(checks.getByTestId('status-checks-toggle'));

    expect(checks.queryByText('Backups configured')).not.toBeInTheDocument();
    expect(
      checks.queryByText('Data loaded from backup')
    ).not.toBeInTheDocument();
  });

  it('says so when no check is reported yet', () => {
    renderCard('never-reconciled-fresh');

    expect(section('checks').getByText('Not reported yet')).toBeInTheDocument();
    expect(
      section('components').getByText('Not reported yet')
    ).toBeInTheDocument();
    expect(section('problems').getByText('None')).toBeInTheDocument();
  });

  it('shows the operation that is running and since when', () => {
    renderCard('rolling-upgrade', upgrading());

    const operation = screen.getByTestId('status-operation-Upgrading');
    expect(operation).toHaveTextContent('Upgrade');
    expect(operation).toHaveTextContent('started 4 min ago');
  });

  it('shows how an operation that is over ended, and when', () => {
    renderCard('rolling-upgrade', upgrading());

    const operation = within(screen.getByTestId('status-finished')).getByTestId(
      'status-operation-StorageResizing'
    );
    expect(operation).toHaveTextContent('Storage resize');
    expect(operation).toHaveTextContent('Succeeded');
    expect(operation).toHaveTextContent('finished 3 h ago');
  });

  it('leaves the finished operations out when the status remembers none', () => {
    renderCard('healthy');

    expect(screen.queryByTestId('status-finished')).not.toBeInTheDocument();
    expect(section('in-progress').getByText('Nothing')).toBeInTheDocument();
  });

  it('shows a notice without making it a problem', () => {
    renderCard('rolling-upgrade', upgrading());

    const notice = screen.getByTestId('status-notice-MaintenancePending');
    expect(notice).toHaveTextContent('Maintenance pending');
    expect(notice).toHaveTextContent('1 action awaits approval');
    expect(section('problems').getByText('None')).toBeInTheDocument();
  });

  it('keeps the raw conditions folded until asked for', () => {
    renderCard('rolling-upgrade', upgrading());
    const toggle = screen.getByTestId('status-conditions-toggle');

    expect(toggle).toHaveTextContent('All conditions (14)');
    expect(
      screen.queryByTestId('status-conditions-table')
    ).not.toBeInTheDocument();

    fireEvent.click(toggle);

    expect(screen.getByTestId('status-conditions-table')).toBeInTheDocument();
    expect(screen.getAllByTestId(/^status-condition-/)).toHaveLength(14);
  });

  it('lists every condition as reported, with its generation and transition', () => {
    renderCard('rolling-upgrade', upgrading());
    fireEvent.click(screen.getByTestId('status-conditions-toggle'));

    const row = screen.getByTestId('status-condition-Upgrading');
    expect(
      within(row)
        .getAllByRole('cell')
        .map((cell) => cell.textContent)
    ).toEqual([
      'Upgrading',
      'True',
      'MinorVersionRolling',
      'Rolling out version 8.0.12 (1 of 3 pods updated)',
      expect.stringMatching(/^01\/03\/2026 at \d\d:56$/),
      '5',
    ]);
  });

  it('lists a condition type it does not know with the others, and nowhere else', () => {
    renderCard('rolling-upgrade', upgrading());

    expect(screen.queryByText('Moving chunks')).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId('status-conditions-toggle'));

    expect(
      screen.getByTestId('status-condition-example.com/ShardsBalanced')
    ).toHaveTextContent('Moving chunks');
  });

  it('warns that the details describe the spec that was replaced', () => {
    renderCard('spec-edited-not-observed');

    expect(screen.getByTestId('status-card-status')).toHaveTextContent(
      'Up \u00b7 Applying changes'
    );
    expect(screen.getByTestId('status-stale')).toBeInTheDocument();
  });

  it('has no such warning for a status that is current', () => {
    renderCard('healthy');

    expect(screen.queryByTestId('status-stale')).not.toBeInTheDocument();
  });
});
