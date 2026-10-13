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
import { AffinityPriority } from 'shared-types/affinity.types';
import EditableItem from 'components/editable-item';
import { AffinityGroup } from '../affinity-group.types';
import { GroupSummary } from '../group-summary';
import { Messages } from '../affinity-rule-editor.messages';
import { CARD_GAP, RAIL_GUTTER } from './affinity-group-list.constants';
import {
  groupIntoOrRuns,
  IndexedGroup,
  splitByPriority,
} from './affinity-group-list.utils';

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
}: AffinityGroupListProps) => {
  const byPriority = splitByPriority(groups);
  const runsByPriority = {
    [AffinityPriority.Required]: groupIntoOrRuns(
      byPriority[AffinityPriority.Required]
    ),
    [AffinityPriority.Preferred]: groupIntoOrRuns(
      byPriority[AffinityPriority.Preferred]
    ),
  };
  // Once a rail exists, every card shares its gutter so all left edges align.
  const gutter = Object.values(runsByPriority).some((runs) =>
    runs.some((run) => run.length > 1)
  )
    ? RAIL_GUTTER
    : 0;

  const renderCard = (entry: IndexedGroup) => (
    <EditableItem
      key={entry.index}
      dataTestId={`affinity-group-${entry.index}`}
      children={<GroupSummary group={entry.group} />}
      editButtonProps={
        actions && { onClick: () => actions.onEdit(entry.index) }
      }
      deleteButtonProps={
        actions && { onClick: () => actions.onRemove(entry.index) }
      }
      // Info-alert look: a light-blue outline + tint per group,
      // instead of piling up neutral nested borders.
      paperProps={{
        sx: {
          // Spacing comes from the surrounding Stack, not the item's own mt.
          mt: 0,
          bgcolor: (theme) => alpha(theme.palette.info.main, 0.04),
          borderColor: (theme) => alpha(theme.palette.info.main, 0.2),
        },
      }}
    />
  );

  return (
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
        <Stack spacing={2.5}>
          {[AffinityPriority.Required, AffinityPriority.Preferred].map(
            (priority) =>
              runsByPriority[priority].length > 0 && (
                <Stack key={priority} spacing={1}>
                  <Typography
                    variant="caption"
                    sx={{ color: 'text.secondary', fontWeight: 600 }}
                  >
                    {Messages.priorityHeading[priority]}
                  </Typography>
                  <Stack spacing={CARD_GAP}>
                    {runsByPriority[priority].map((run) =>
                      run.length === 1 ? (
                        <Box key={run[0].index} sx={{ pl: gutter }}>
                          {renderCard(run[0])}
                        </Box>
                      ) : (
                        // A left rail binds the alternatives; OR sits on it.
                        <Box
                          key={run[0].index}
                          data-testid="affinity-or-group"
                          sx={{
                            position: 'relative',
                            pl: RAIL_GUTTER,
                            '&::before': {
                              content: '""',
                              position: 'absolute',
                              left: 0,
                              top: 8,
                              bottom: 8,
                              width: 2,
                              borderRadius: 1,
                              bgcolor: (theme) =>
                                alpha(theme.palette.primary.main, 0.4),
                            },
                          }}
                        >
                          <Stack spacing={CARD_GAP}>
                            {run.map((entry, position) => (
                              <Box
                                key={entry.index}
                                sx={{ position: 'relative' }}
                              >
                                {position > 0 && (
                                  // Centered on the rail, inside the gap above.
                                  <Typography
                                    variant="caption"
                                    sx={{
                                      position: 'absolute',
                                      top: (theme) =>
                                        `calc(-${theme.spacing(CARD_GAP)} / 2)`,
                                      left: (theme) =>
                                        `calc(1px - ${theme.spacing(RAIL_GUTTER)})`,
                                      transform: 'translate(-50%, -50%)',
                                      px: 0.5,
                                      lineHeight: 1.5,
                                      bgcolor: 'background.paper',
                                      fontWeight: 700,
                                      color: 'primary.main',
                                    }}
                                  >
                                    {Messages.or}
                                  </Typography>
                                )}
                                {renderCard(entry)}
                              </Box>
                            ))}
                          </Stack>
                        </Box>
                      )
                    )}
                  </Stack>
                </Stack>
              )
          )}
        </Stack>
      )}
    </Stack>
  );
};
