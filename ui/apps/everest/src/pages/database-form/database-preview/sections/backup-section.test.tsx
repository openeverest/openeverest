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
import { TestWrapper } from 'utils/test';
import { PreviewBackupSection } from './backup-section';

// PITR on a storage that has no schedule.
const pitrOnly = {
  provider: 'percona-server-mongodb',
  dbName: 'my-db',
  k8sNamespace: 'team-a',
  topology: { type: 'replicaSet' },
  spec: { providerRef: { name: 'percona-server-mongodb' } },
  backup: {
    schedules: [],
    classRef: { name: 'psmdb-backup-class' },
    pitr: { 'bs-team-a': { enabled: true } },
  },
};

const renderSection = (presetId?: string) =>
  render(
    <TestWrapper>
      <PreviewBackupSection {...pitrOnly} presetId={presetId} />
    </TestWrapper>
  );

describe('PreviewBackupSection', () => {
  // A preset is deployed as is, so its PITR runs even with nothing scheduled.
  it('shows the PITR of a preset that schedules no backups', () => {
    renderSection('namespace/team-a/psmdb-default');

    expect(screen.getByText('No backup schedules')).toBeInTheDocument();
    expect(screen.getByText('PITR: enabled on bs-team-a')).toBeInTheDocument();
  });

  // Without a preset the wizard builds the spec itself and attaches PITR only
  // to storages that have a schedule, so there is nothing to promise here.
  it('does not promise PITR the wizard would not create', () => {
    renderSection();

    expect(screen.getByText('No backup schedules')).toBeInTheDocument();
    expect(screen.queryByText(/PITR/)).not.toBeInTheDocument();
  });
});
