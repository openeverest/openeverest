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
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  alpha,
  Box,
  Button,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { useController, useFormContext } from 'react-hook-form';
import { Affinity } from 'shared-types/affinity.types';
import EditableItem from 'components/editable-item';
import { WidgetRendererProps } from 'components/ui-generator/ui-generator.types';
import { AffinityGroup } from './affinity-group.types';
import { affinityToGroups, groupsToAffinity } from './affinity-group-converter';
import { GroupSummary } from './group-summary';
import { GroupEditorDialog } from './group-editor-dialog/group-editor-dialog';
import { Messages } from './affinity-rule-editor.messages';

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
      {/* Single policy in the section is pinned open; multiple policies (future
          tolerations / node selector) would be collapsible accordions. */}
      <Accordion
        expanded
        disableGutters
        variant="outlined"
        sx={{ '&::before': { display: 'none' } }}
      >
        <AccordionSummary
          sx={{
            cursor: 'default',
            '& .MuiAccordionSummary-content': {
              alignItems: 'center',
              justifyContent: 'space-between',
              mr: 1,
            },
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
            <Typography variant="sectionHeading">{Messages.label}</Typography>
            <Tooltip title={Messages.groupsInfo} placement="right" arrow>
              <InfoOutlinedIcon
                sx={{ fontSize: 18, color: 'text.secondary' }}
              />
            </Tooltip>
          </Box>
          <Button size="small" startIcon={<AddIcon />} onClick={openAdd}>
            {Messages.addGroup}
          </Button>
        </AccordionSummary>
        <AccordionDetails>
          <Stack spacing={1}>
            {groups.length === 0 ? (
              <EditableItem
                dataTestId="empty"
                children={
                  <Typography variant="body1">{Messages.empty}</Typography>
                }
              />
            ) : (
              groups.map((group, index) => (
                <EditableItem
                  key={index}
                  dataTestId={`affinity-group-${index}`}
                  children={<GroupSummary group={group} />}
                  editButtonProps={{ onClick: () => openEdit(index) }}
                  deleteButtonProps={{ onClick: () => remove(index) }}
                  // Info-alert look: a light-blue outline + tint per group,
                  // instead of piling up neutral nested borders.
                  paperProps={{
                    sx: {
                      bgcolor: (theme) => alpha(theme.palette.info.main, 0.04),
                      borderColor: (theme) =>
                        alpha(theme.palette.info.main, 0.2),
                    },
                  }}
                />
              ))
            )}
          </Stack>
        </AccordionDetails>
      </Accordion>

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
