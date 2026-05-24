// everest
// Copyright (C) 2023 Percona LLC
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

import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import { Box, Button, Typography } from '@mui/material';
import { Link as RouterLink } from 'react-router-dom';

export type EmptyStateWithLinkProps = {
  /** Main explanatory message shown below the warning icon */
  message: string;
  /** Label for the navigation CTA */
  linkLabel: string;
  /** react-router `to` target — a route, not a cluster-state mutation */
  to: string;
  /** Optional data-testid for e2e targeting */
  dataTestId?: string;
};

/**
 * Read-only empty state with a navigational link.
 *
 * Follows the "Overview = observe, Tabs = act" principle: the card itself
 * stays observation-only.  Routing to a settings or action page is a
 * navigation concern, not a cluster-state mutation, so a `<Link>` is the
 * correct primitive here.
 *
 * Reusable for any overview card that needs to guide the user to a
 * configuration page (backup storages, monitoring endpoints, etc.)
 */
const EmptyStateWithLink = ({
  message,
  linkLabel,
  to,
  dataTestId = 'empty-state-with-link',
}: EmptyStateWithLinkProps) => {
  return (
    <Box
      data-testid={dataTestId}
      sx={{
        display: 'flex',
        py: 6,
        px: 0,
        flexDirection: 'column',
        alignItems: 'center',
        gap: 1,
        alignSelf: 'stretch',
      }}
    >
      <Box sx={{ fontSize: '100px', lineHeight: 0 }}>
        <WarningAmberIcon fontSize="inherit" />
      </Box>
      <Typography variant="body1">{message}</Typography>
      <Button
        sx={{ my: 4 }}
        variant="contained"
        component={RouterLink}
        to={to}
        data-testid={`${dataTestId}-cta`}
      >
        {linkLabel}
      </Button>
    </Box>
  );
};

export default EmptyStateWithLink;
