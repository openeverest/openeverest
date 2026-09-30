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

import { useMemo, useState } from 'react';
import { useController, useFormContext } from 'react-hook-form';
import { Affinity } from 'shared-types/affinity.types';
import { WidgetRendererProps } from 'components/ui-generator/ui-generator.types';
import { AffinityGroup } from './affinity-group.types';
import { affinityToGroups, groupsToAffinity } from './affinity-group-converter';
import { AffinityGroupList } from './affinity-group-list';
import { GroupEditorDialog } from './group-editor-dialog/group-editor-dialog';

// The field path is spec.components.<component>.schedulingPolicy.affinity.
const componentFromName = (name: string): string | undefined =>
  name.match(/components\.([^.]+)\./)?.[1];

// The form field holds the Kubernetes Affinity itself (what is submitted). The
// editor derives the grouped UI model for display/editing and converts back on
// every change, so the field always carries a valid payload.
export const AffinityRuleEditor = ({ name }: WidgetRendererProps) => {
  const { control } = useFormContext();
  const { field } = useController({ name, control });
  const affinity: Affinity = field.value ?? {};
  const groups = useMemo(() => affinityToGroups(affinity), [field.value]);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | undefined>(
    undefined
  );

  const commit = (nextGroups: AffinityGroup[]) =>
    field.onChange(groupsToAffinity(nextGroups));

  const openAdd = () => {
    setEditingIndex(undefined);
    setDialogOpen(true);
  };

  const openEdit = (index: number) => {
    setEditingIndex(index);
    setDialogOpen(true);
  };

  const remove = (index: number) =>
    commit(groups.filter((_, i) => i !== index));

  const save = (group: AffinityGroup) => {
    commit(
      editingIndex === undefined
        ? [...groups, group]
        : groups.map((existing, i) => (i === editingIndex ? group : existing))
    );
    setDialogOpen(false);
  };

  return (
    <>
      <AffinityGroupList
        groups={groups}
        actions={{ onAdd: openAdd, onEdit: openEdit, onRemove: remove }}
      />

      {dialogOpen && (
        <GroupEditorDialog
          isOpen
          group={editingIndex !== undefined ? groups[editingIndex] : undefined}
          component={componentFromName(name)}
          onClose={() => setDialogOpen(false)}
          onSubmit={save}
        />
      )}
    </>
  );
};
