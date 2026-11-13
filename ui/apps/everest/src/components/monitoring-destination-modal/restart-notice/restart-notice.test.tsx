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

import { ReactNode } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { RestartNotice } from './restart-notice';

const mocks = vi.hoisted((): { readableInstances: string[] } => ({
  readableInstances: [],
}));

vi.mock('../hooks/use-destination-instances', () => ({
  useDestinationInstances: () => mocks.readableInstances,
}));

const destination = (attachedInstances: number): MonitoringDestination => ({
  metadata: { name: 'pmm-prod', namespace: 'team-a' },
  spec: { classRef: { name: 'pmm' } },
  status: { attachedInstances },
});

const monitoringClass = (
  disruption: 'None' | 'RollingRestart'
): MonitoringClass => ({
  metadata: { name: 'pmm' },
  spec: {
    controllerName: 'openeverest.io/monitoring-pmm',
    authModes: [{ name: 'Anonymous' }],
  },
  status: {
    providers: [
      { name: 'percona-xtradb-cluster', disruption: 'None' },
      { name: 'percona-server-mongodb', disruption },
    ],
  },
});

const Form = ({ children }: { children: ReactNode }) => {
  const methods = useForm({
    defaultValues: {
      spec: { parameters: { url: 'https://pmm' } },
      fallbackParameters: {},
      credentials: { token: '' },
      insecureSkipVerify: false,
    },
  });
  return (
    <FormProvider {...methods}>
      <input aria-label="url" {...methods.register('spec.parameters.url')} />
      <input aria-label="token" {...methods.register('credentials.token')} />
      <input
        type="checkbox"
        aria-label="skip TLS"
        {...methods.register('insecureSkipVerify')}
      />
      {children}
    </FormProvider>
  );
};

const renderNotice = ({
  attached = 0,
  readable = [],
  disruption = 'RollingRestart',
}: {
  attached?: number;
  readable?: string[];
  disruption?: 'None' | 'RollingRestart';
}) => {
  mocks.readableInstances = readable;
  render(
    <Form>
      <RestartNotice
        destination={destination(attached)}
        monitoringClass={monitoringClass(disruption)}
      />
    </Form>
  );
};

const changeUrl = (url: string) =>
  fireEvent.change(screen.getByLabelText('url'), { target: { value: url } });
const notice = () => screen.queryByTestId('monitoring-destination-restart');

describe('RestartNotice', () => {
  it('warns only while a parameter differs from the saved destination', () => {
    renderNotice({ attached: 2 });
    expect(notice()).not.toBeInTheDocument();

    changeUrl('https://pmm-new');
    expect(notice()).toHaveTextContent(
      'may restart the pods of the 2 instances that send to this destination'
    );

    changeUrl('https://pmm');
    expect(notice()).not.toBeInTheDocument();
  });

  it('warns when new credentials are entered', () => {
    renderNotice({ attached: 1 });
    fireEvent.change(screen.getByLabelText('token'), {
      target: { value: 'glsa_new' },
    });
    expect(notice()).toBeInTheDocument();
  });

  it('stays quiet when only TLS changes', () => {
    renderNotice({ attached: 2 });
    fireEvent.click(screen.getByLabelText('skip TLS'));
    expect(notice()).not.toBeInTheDocument();
  });

  it.each([
    {
      name: 'Instances the user may not read',
      attached: 3,
      readable: [],
      expected: 'the 3 instances',
    },
    {
      name: 'an attachment the status has not counted yet',
      attached: 0,
      readable: ['orders'],
      expected: 'the instance that sends',
    },
  ])('counts $name', ({ attached, readable, expected }) => {
    renderNotice({ attached, readable });
    changeUrl('https://pmm-new');
    expect(notice()).toHaveTextContent(expected);
  });

  it.each<{
    name: string;
    attached?: number;
    disruption?: 'None' | 'RollingRestart';
  }>([
    { name: 'nothing is attached', attached: 0 },
    { name: 'the class does not roll the pods', disruption: 'None' },
  ])('stays quiet when $name', ({ attached = 1, disruption }) => {
    renderNotice({ attached, disruption });
    changeUrl('https://pmm-new');
    expect(notice()).not.toBeInTheDocument();
  });
});
