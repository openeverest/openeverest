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

import KeyboardArrowDownOutlined from '@mui/icons-material/KeyboardArrowDownOutlined';
import KeyboardArrowUpOutlined from '@mui/icons-material/KeyboardArrowUpOutlined';
import { Alert, AlertTitle, Box, Collapse, Link } from '@mui/material';
import { CodeCopyBlock } from '@percona/ui-lib';
import { useState } from 'react';
import type {
  InstanceCondition,
  PodsAlertReason,
} from 'shared-types/instance.types';
import { Messages } from './pods-alert.messages';

interface PodsAlertProps {
  condition: InstanceCondition;
}

export const PodsAlert = ({ condition }: PodsAlertProps) => {
  const [detailsOpen, setDetailsOpen] = useState(false);
  const { title, hint } = Messages.alerts[condition.reason as PodsAlertReason];

  return (
    <Alert severity="warning" sx={{ my: 1 }} data-testid="pods-alert">
      <AlertTitle>{title}</AlertTitle>
      {hint}
      <Box>
        <Link
          component="button"
          color="inherit"
          underline="hover"
          aria-expanded={detailsOpen}
          onClick={() => setDetailsOpen((open) => !open)}
          sx={{
            mt: 1,
            display: 'inline-flex',
            alignItems: 'center',
            fontWeight: 600,
          }}
        >
          {detailsOpen ? Messages.hideDetails : Messages.showDetails}
          {detailsOpen ? (
            <KeyboardArrowUpOutlined fontSize="small" />
          ) : (
            <KeyboardArrowDownOutlined fontSize="small" />
          )}
        </Link>
      </Box>
      <Collapse in={detailsOpen} unmountOnExit>
        <CodeCopyBlock message={condition.message} severity="warning" />
      </Collapse>
    </Alert>
  );
};
