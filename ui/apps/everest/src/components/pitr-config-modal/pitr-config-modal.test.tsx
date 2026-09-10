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

import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BackupClass } from 'shared-types/backups.types';
import { PitrConfigModal } from './pitr-config-modal';

// Mirrors the PXC provider's pitr uiSchema: relative `parameters.<leaf>` paths.
const pxcClass = {
  metadata: { name: 'pxc-managed' },
  spec: {
    uiSchema: {
      pitr: {
        label: 'PITR Configuration',
        componentsOrder: ['timeBetweenUploads', 'timeoutSeconds'],
        components: {
          timeBetweenUploads: {
            uiType: 'number',
            path: 'parameters.timeBetweenUploads',
            fieldParams: {
              label: 'Upload interval (seconds)',
              defaultValue: 60,
            },
            validation: { min: 1 },
          },
          timeoutSeconds: {
            uiType: 'number',
            path: 'parameters.timeoutSeconds',
            fieldParams: {
              label: 'Upload timeout (seconds)',
              defaultValue: 3600,
            },
            validation: { min: 1 },
          },
        },
      },
    },
  },
} as unknown as BackupClass;

const renderModal = (
  overrides: Partial<React.ComponentProps<typeof PitrConfigModal>> = {}
) => {
  const onSubmit = vi.fn();
  render(
    <PitrConfigModal
      open
      storageName="s3-main"
      backupClass={pxcClass}
      onClose={vi.fn()}
      onSubmit={onSubmit}
      {...overrides}
    />
  );
  return { onSubmit };
};

describe('PitrConfigModal', () => {
  it('renders the provider pitr fields prefilled with schema defaults', () => {
    renderModal();

    expect(screen.getByLabelText(/Upload interval/i)).toHaveValue(60);
    expect(screen.getByLabelText(/Upload timeout/i)).toHaveValue(3600);
  });

  it('prefers previously saved parameters over schema defaults', () => {
    renderModal({ currentParameters: { timeBetweenUploads: 120 } });

    expect(screen.getByLabelText(/Upload interval/i)).toHaveValue(120);
    expect(screen.getByLabelText(/Upload timeout/i)).toHaveValue(3600);
  });

  it('packs the pitr fields into parameters on save', async () => {
    const { onSubmit } = renderModal();

    const saveButton = screen.getByRole('button', { name: /save/i });
    await waitFor(() => expect(saveButton).toBeEnabled());
    fireEvent.click(saveButton);

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit.mock.calls[0][0]).toEqual({
      timeBetweenUploads: 60,
      timeoutSeconds: 3600,
    });
  });
});
