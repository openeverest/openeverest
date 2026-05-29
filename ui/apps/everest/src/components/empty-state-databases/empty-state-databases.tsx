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

import { Box, Link, Typography } from '@mui/material';
import CreateDbButton from 'components/create-db-button/create-db-button';
import EmptyState from 'components/empty-state';
import { Messages } from './messages';

const EmptyStateDatabases = ({
  showCreationButton,
  hasCreatePermission,
}: {
  showCreationButton: boolean;
  hasCreatePermission: boolean;
}) => {
  return (
    <>
      <EmptyState
        contentSlot={
          <>
            <Typography>{Messages.noDbClusters}</Typography>
            {hasCreatePermission ? (
              <Typography> {Messages.createToStart} </Typography>
            ) : (
              <>
                <Typography>{Messages.noPermissions}</Typography>
                <Typography>
                  Click{' '}
                  <Link
                    target="_blank"
                    rel="noopener"
                    href="https://openeverest.io/documentation/current/administer/rbac.html"
                  >
                    here
                  </Link>{' '}
                  to learn how to get permissions.
                </Typography>
              </>
            )}
          </>
        }
        showCreationButton={showCreationButton}
        buttonSlot={
          <Box display="flex" mb={1}>
            {/* <CreateDbButton createFromImport /> */}
            <CreateDbButton />
          </Box>
        }
      />
    </>
  );
};

export default EmptyStateDatabases;
