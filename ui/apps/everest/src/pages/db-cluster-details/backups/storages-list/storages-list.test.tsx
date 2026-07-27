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
import { StoragesList } from './storages-list';
import { ScheduleModalContext } from '../backups.context';
import { ScheduleModalContextType } from '../backups.types';
import { Instance } from 'shared-types/api.types';
import { WizardMode } from 'shared-types/wizard.types';

const buildInstance = (
  storages: NonNullable<
    NonNullable<NonNullable<Instance['spec']>['backup']>['storages']
  >
): Instance =>
  ({
    spec: {
      backup: {
        enabled: true,
        classRef: { name: 'psmdb-managed' },
        storages,
      },
    },
  }) as unknown as Instance;

const renderWithContext = (instance: Instance) => {
  const contextValue: ScheduleModalContextType = {
    instance,
    mode: WizardMode.New,
    setMode: vi.fn(),
    selectedScheduleName: '',
    setSelectedScheduleName: vi.fn(),
    openScheduleModal: false,
    setOpenScheduleModal: vi.fn(),
    openOnDemandModal: false,
    setOpenOnDemandModal: vi.fn(),
  };

  return render(
    <ScheduleModalContext.Provider value={contextValue}>
      <StoragesList />
    </ScheduleModalContext.Provider>
  );
};

describe('StoragesList', () => {
  it('renders a row per instance storage', () => {
    renderWithContext(
      buildInstance([
        { storageRef: { name: 's3-prod' } },
        { storageRef: { name: 's3-archive' } },
      ])
    );

    expect(screen.getByText('s3-prod')).toBeInTheDocument();
    expect(screen.getByText('s3-archive')).toBeInTheDocument();
  });

  it('renders nothing when the instance has no storages', () => {
    const { container } = renderWithContext(buildInstance([]));

    expect(container).toBeEmptyDOMElement();
  });
});
