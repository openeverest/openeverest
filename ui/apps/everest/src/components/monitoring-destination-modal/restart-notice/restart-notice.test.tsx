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
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { RestartNotice } from './restart-notice';

const mocks = vi.hoisted((): { instances: string[] | undefined } => ({
  instances: undefined,
}));

vi.mock('../hooks/use-destination-instances', () => ({
  useDestinationInstances: () => mocks.instances,
}));

const destination: MonitoringDestination = {
  metadata: { name: 'pmm-prod', namespace: 'team-a' },
  spec: { classRef: { name: 'pmm' } },
};

const monitoringClass = (
  disruption: 'None' | 'RollingRestart'
): MonitoringClass => ({
  metadata: { name: 'pmm' },
  spec: { authModes: [{ name: 'Anonymous' }] },
  status: {
    providers: [
      { name: 'percona-xtradb-cluster', disruption: 'None' },
      { name: 'percona-server-mongodb', disruption },
    ],
  },
});

const renderNotice = (
  instances: string[] | undefined,
  disruption: 'None' | 'RollingRestart' = 'RollingRestart'
) => {
  mocks.instances = instances;
  render(
    <RestartNotice
      destination={destination}
      monitoringClass={monitoringClass(disruption)}
    />
  );
};

const notice = () => screen.queryByTestId('monitoring-destination-restart');

describe('RestartNotice', () => {
  it('counts the instances that restart when any provider rolls the pods', () => {
    renderNotice(['orders', 'billing']);
    expect(notice()).toHaveTextContent(
      'restarts the pods of the 2 instances that send to this destination'
    );
  });

  it('warns about every instance when the user may not list them', () => {
    renderNotice(undefined);
    expect(notice()).toHaveTextContent(
      'restarts the pods of every instance that sends to this destination'
    );
  });

  it('stays quiet when nothing would restart', () => {
    renderNotice([]);
    expect(notice()).not.toBeInTheDocument();
  });

  it('stays quiet for a class that does not roll the pods', () => {
    renderNotice(['orders'], 'None');
    expect(notice()).not.toBeInTheDocument();
  });
});
