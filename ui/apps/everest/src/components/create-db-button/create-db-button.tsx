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

import { useEffect, useState } from 'react';
import {
  Box,
  Button,
  Drawer,
  IconButton,
  Skeleton,
  Tooltip,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { ArrowDropDownIcon } from '@mui/x-date-pickers/icons';
import { useNavigate } from 'react-router-dom';
import { useProviders } from 'hooks/api/providers';
import type { CreateDbButtonProps } from './create-db-button.types';
import { Messages } from './create-db-button.messages';
import {
  buttonSx,
  drawerBannerSx,
  drawerContainerSx,
  drawerHeaderSx,
  drawerPaperSx,
  REVEAL_DELAY_MS,
  skeletonSx,
} from './create-db-button.constants';
import { DrawerBody } from './drawer-body';

export const CreateDbButton = ({
  createFromImport = false,
}: CreateDbButtonProps) => {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [showDropdownButton, setShowDropdownButton] = useState(false);
  // TODO check how it should work with Providers
  // const { canCreate } = useNamespacePermissionsForResource('database-clusters');

  const { data: providers = [], isLoading: providersLoading } = useProviders();
  const navigate = useNavigate();

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    if (providers.length > 1) {
      event.stopPropagation();
      setDrawerOpen(true);
    } else {
      navigate('/databases/new', {
        state: {
          selectedDbProvider: providers[0],
          showImport: createFromImport,
        },
      });
    }
  };

  const closeDrawer = () => {
    setDrawerOpen(false);
  };

  useEffect(() => {
    let timeoutId: ReturnType<typeof setTimeout> | undefined;

    if (providersLoading) {
      setShowDropdownButton(false);
    } else {
      timeoutId = setTimeout(() => {
        setShowDropdownButton(true);
      }, REVEAL_DELAY_MS);
    }

    return () => {
      if (timeoutId !== undefined) {
        clearTimeout(timeoutId);
      }
    };
  }, [providersLoading]);

  const testIdPrefix = createFromImport ? 'import' : 'add';
  const showTechPreviewTooltip = createFromImport && providers.length === 1;

  const createButton = (
    <Button
      data-testid={`${testIdPrefix}-db-cluster-button`}
      size="small"
      variant={createFromImport ? 'text' : 'contained'}
      sx={buttonSx}
      aria-controls={
        drawerOpen ? `${testIdPrefix}-db-cluster-button-menu` : undefined
      }
      aria-haspopup={providers.length > 1 ? 'dialog' : undefined}
      aria-expanded={
        providers.length > 1 ? (drawerOpen ? 'true' : 'false') : undefined
      }
      onClick={handleClick}
      endIcon={providers.length > 1 && <ArrowDropDownIcon />}
    >
      {createFromImport ? Messages.import : Messages.createDatabase}
    </Button>
  );

  if (providers.length === 0) {
    return null;
  }

  return (
    <Box>
      {showDropdownButton ? (
        showTechPreviewTooltip ? (
          <Tooltip title={Messages.technicalPreview} enterDelay={0}>
            {createButton}
          </Tooltip>
        ) : (
          createButton
        )
      ) : (
        <Skeleton variant="rounded" sx={skeletonSx} />
      )}
      {providers.length > 1 && (
        <Drawer
          data-testid={`${testIdPrefix}-db-cluster-button-menu`}
          anchor="right"
          open={drawerOpen}
          onClose={closeDrawer}
          sx={{ zIndex: (theme) => theme.zIndex.modal }}
          slotProps={{ paper: { sx: drawerPaperSx } }}
        >
          <Box sx={drawerContainerSx}>
            <Box sx={drawerHeaderSx}>
              <Box>
                <Typography variant="h6">
                  {createFromImport
                    ? Messages.importDatabase
                    : Messages.createDatabase}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {Messages.selectProvider}
                </Typography>
              </Box>
              <IconButton
                aria-label={Messages.closeDrawerAriaLabel}
                onClick={closeDrawer}
                size="small"
              >
                <CloseIcon fontSize="small" />
              </IconButton>
            </Box>

            {createFromImport && (
              <Box sx={drawerBannerSx}>
                <Typography variant="caption" color="text.secondary">
                  {Messages.technicalPreview}
                </Typography>
              </Box>
            )}

            <DrawerBody
              providers={providers}
              createFromImport={createFromImport}
              onClose={closeDrawer}
            />
          </Box>
        </Drawer>
      )}
    </Box>
  );
};
