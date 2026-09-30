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

import { alpha, Box, Button, Stack, Tooltip, Typography } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import EditableItem from 'components/editable-item';
import { AffinityGroup } from '../affinity-group.types';
import { GroupSummary } from '../group-summary';
import { Messages } from '../affinity-rule-editor.messages';

interface AffinityGroupListActions {
  onAdd: () => void;
  onEdit: (index: number) => void;
  onRemove: (index: number) => void;
}

interface AffinityGroupListProps {
  groups: AffinityGroup[];
  // Omitted in read-only surfaces: no add / edit / delete affordances.
  actions?: AffinityGroupListActions;
}

export const AffinityGroupList = ({
  groups,
  actions,
}: AffinityGroupListProps) => (
  <Stack spacing={1}>
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        minHeight: 32,
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
        <Typography variant="sectionHeading">{Messages.label}</Typography>
        <Tooltip title={Messages.groupsInfo} placement="right" arrow>
          <InfoOutlinedIcon sx={{ fontSize: 18, color: 'text.secondary' }} />
        </Tooltip>
      </Box>
      {actions && (
        <Button size="small" startIcon={<AddIcon />} onClick={actions.onAdd}>
          {Messages.addGroup}
        </Button>
      )}
    </Box>
    {groups.length === 0 ? (
      <EditableItem
        dataTestId="empty"
        children={
          <Typography variant="body1">
            {actions ? Messages.empty : Messages.emptyReadOnly}
          </Typography>
        }
      />
    ) : (
      groups.map((group, index) => (
        <EditableItem
          key={index}
          dataTestId={`affinity-group-${index}`}
          children={<GroupSummary group={group} />}
          editButtonProps={actions && { onClick: () => actions.onEdit(index) }}
          deleteButtonProps={
            actions && { onClick: () => actions.onRemove(index) }
          }
          // Info-alert look: a light-blue outline + tint per group,
          // instead of piling up neutral nested borders.
          paperProps={{
            sx: {
              bgcolor: (theme) => alpha(theme.palette.info.main, 0.04),
              borderColor: (theme) => alpha(theme.palette.info.main, 0.2),
            },
          }}
        />
      ))
    )}
  </Stack>
);
