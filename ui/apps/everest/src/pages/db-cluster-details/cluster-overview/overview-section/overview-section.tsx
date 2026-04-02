// everest
// Copyright (C) 2023 Percona LLC
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

// TODO refactor and move to the components library

import {
  Grid,
  Stack,
  Typography,
  Divider,
  Box,
  Button,
  Tooltip,
} from '@mui/material';
import { LoadableChildren } from '@percona/ui-lib';
import { OverviewSectionProps } from './overview-section.types';
import { Messages } from '../cluster-overview.messages';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';

export const OverviewSection = ({
  title,
  loading,
  children,
  dataTestId,
  editable,
  actionButtonProps,
  showTooltip = false,
  disabledEditTooltipText = '',
  editText = Messages.actions.edit,
}: OverviewSectionProps) => (
  <Grid
    item
    xs={6}
    data-testid={
      dataTestId ? `${dataTestId}-overview-section` : 'overview-section'
    }
  >
    <Stack>
      {(title || actionButtonProps) && (
        <>
          <Stack
            direction="row"
            justifyContent="space-between"
            alignItems="flex-end"
          >
            <Typography color="text.primary" variant="sectionHeading">
              {title}
            </Typography>
            {actionButtonProps && (
              <Tooltip title={showTooltip ? disabledEditTooltipText : ''}>
                <Box>
                  <Button
                    size="small"
                    disabled={!editable}
                    startIcon={<EditOutlinedIcon />}
                    {...actionButtonProps}
                  >
                    {editText}
                  </Button>
                </Box>
              </Tooltip>
            )}
          </Stack>
          <Divider sx={{ mt: 0.25 }} />
        </>
      )}
      <LoadableChildren loading={loading}>
        <Box sx={{ mt: title || actionButtonProps ? 1 : 0 }}>{children}</Box>
      </LoadableChildren>
    </Stack>
  </Grid>
);

export default OverviewSection;
