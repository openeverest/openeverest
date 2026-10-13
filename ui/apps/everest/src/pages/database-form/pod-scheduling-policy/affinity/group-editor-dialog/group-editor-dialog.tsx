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

import { FormDialog } from 'components/form-dialog';
import { affinityGroupSchema } from '../affinity-group-schema';
import { AffinityGroup } from '../affinity-group.types';
import { GroupForm } from './group-form';
import { emptyGroup } from './group-editor-dialog.utils';
import { Messages } from './group-editor-dialog.messages';

interface GroupEditorDialogProps {
  isOpen: boolean;
  group?: AffinityGroup;
  component?: string;
  onClose: () => void;
  onSubmit: (group: AffinityGroup) => void;
}

export const GroupEditorDialog = ({
  isOpen,
  group,
  component,
  onClose,
  onSubmit,
}: GroupEditorDialogProps) => {
  const isEditing = !!group;
  const title = component
    ? isEditing
      ? Messages.editGroupFor(component)
      : Messages.addGroupTo(component)
    : isEditing
      ? Messages.editGroup
      : Messages.addGroup;

  return (
    <FormDialog<AffinityGroup>
      schema={affinityGroupSchema}
      isOpen={isOpen}
      closeModal={onClose}
      onSubmit={onSubmit}
      validationMode="onTouched"
      headerMessage={title}
      description={Messages.description}
      submitMessage={isEditing ? Messages.save : Messages.add}
      defaultValues={group ?? emptyGroup()}
      size="XXL"
      dataTestId="affinity-group"
    >
      <GroupForm />
    </FormDialog>
  );
};
