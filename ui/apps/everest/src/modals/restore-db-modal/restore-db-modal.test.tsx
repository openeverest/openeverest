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
import RestoreDbModal from './restore-db-modal';
import { Messages } from './restore-db-modal.messages';

const mockUseRestoreDbModal = vi.fn();
const mockFormDialog = vi.fn();
const mockModalContent = vi.fn();

vi.mock('./useRestoreDbModal', () => ({
  useRestoreDbModal: (...args: unknown[]) => mockUseRestoreDbModal(...args),
}));

vi.mock('components/form-dialog', () => ({
  FormDialog: (props: {
    headerMessage: string;
    submitMessage: string;
    children: React.ReactNode;
  }) => {
    mockFormDialog(props);
    return <div data-testid="form-dialog-mock">{props.children}</div>;
  },
}));

vi.mock('./modal-content', () => ({
  ModalContent: (props: { header: string }) => {
    mockModalContent(props);
    return <div data-testid="modal-content-mock">{props.header}</div>;
  },
}));

describe('RestoreDbModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseRestoreDbModal.mockReturnValue({
      isLoading: false,
      succeededBackups: [],
      pitrStorages: [],
      submitting: false,
      restoreSchema: {
        safeParse: () => ({ success: true }),
      },
      formDefaultValues: {},
      onSubmit: vi.fn(),
    });
  });

  it('passes restore mode labels to FormDialog', () => {
    render(
      <RestoreDbModal
        isOpen
        closeModal={vi.fn()}
        instanceName="source-db"
        namespace="ns-a"
      />
    );

    expect(mockFormDialog).toHaveBeenCalledWith(
      expect.objectContaining({
        headerMessage: Messages.headerMessage,
        submitMessage: Messages.restore,
      })
    );
    expect(screen.getByTestId('modal-content-mock')).toHaveTextContent(
      Messages.subHead
    );
  });

  it('passes create mode labels and preselected backup to hook', () => {
    render(
      <RestoreDbModal
        isOpen
        closeModal={vi.fn()}
        instanceName="source-db"
        namespace="ns-a"
        isNewClusterMode
        preselectedBackupName="daily-1"
      />
    );

    expect(mockUseRestoreDbModal).toHaveBeenCalledWith({
      instanceName: 'source-db',
      namespace: 'ns-a',
      isNewClusterMode: true,
      preselectedBackupName: 'daily-1',
      closeModal: expect.any(Function),
    });
    expect(mockFormDialog).toHaveBeenCalledWith(
      expect.objectContaining({
        headerMessage: Messages.headerMessageCreate,
        submitMessage: Messages.create,
      })
    );
    expect(screen.getByTestId('modal-content-mock')).toHaveTextContent(
      Messages.subHeadCreate
    );
  });
});
