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
  Avatar,
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
import { Link, useNavigate } from 'react-router-dom';
import { useExtensionCatalog } from 'hooks/api/extension-catalog';
import { useProviders } from 'hooks/api/providers';

export const CreateDbButton = ({
  createFromImport = false,
}: {
  createFromImport?: boolean;
}) => {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [showDropdownButton, setShowDropdownButton] = useState(false);
  // TODO check how it should work with Providers
  // const { canCreate } = useNamespacePermissionsForResource('database-clusters');

  // const { data: availableDbImporters } = useDataImporters();
  // TODO check how it should work with Providers
  // const supportedEngineTypesForImport = new Set(
  //   availableDbImporters?.items
  //     .map((importer) => importer.spec.supportedEngines)
  //     .flat()
  // );

  const { data: providers = [], isLoading: providersLoading } = useProviders();
  const { getProviderMeta } = useExtensionCatalog();
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
  const closeMenu = () => {
    setDrawerOpen(false);
  };

  useEffect(() => {
    let timeoutId: ReturnType<typeof setTimeout> | undefined;

    if (providersLoading) {
      setShowDropdownButton(false);
    } else {
      timeoutId = setTimeout(() => {
        setShowDropdownButton(true);
      }, 300);
    }

    return () => {
      if (timeoutId !== undefined) {
        clearTimeout(timeoutId);
      }
    };
  }, [providersLoading]);

  const buttonStyle = { display: 'flex', minHeight: '34px', width: '165px' };
  const skeletonStyle = {
    ...buttonStyle,
    borderRadius: '128px',
  };
  const showTechPreviewTooltip = createFromImport && providers.length === 1;

  const techPreviewText = 'Technical Preview';

  const createButton = (
    <Button
      data-testid={`${createFromImport ? 'import' : 'add'}-db-cluster-button`}
      size="small"
      variant={createFromImport ? 'text' : 'contained'}
      sx={buttonStyle}
      aria-controls={
        drawerOpen
          ? `${createFromImport ? 'import' : 'add'}-db-cluster-button-menu`
          : undefined
      }
      aria-haspopup="true"
      aria-expanded={drawerOpen ? 'true' : undefined}
      onClick={handleClick}
      endIcon={providers.length > 1 && <ArrowDropDownIcon />}
    >
      {createFromImport ? 'Import' : 'Create database'}
    </Button>
  );

  return providers.length > 0 ? (
    <Box>
      {showDropdownButton ? (
        showTechPreviewTooltip ? (
          <Tooltip title={techPreviewText} enterDelay={0}>
            {createButton}
          </Tooltip>
        ) : (
          createButton
        )
      ) : (
        <Skeleton variant="rounded" sx={skeletonStyle} />
      )}
      {providers.length > 1 && (
        <Drawer
          data-testid={`${
            createFromImport ? 'import' : 'add'
          }-db-cluster-button-menu`}
          anchor="right"
          open={drawerOpen}
          onClose={closeMenu}
          sx={{ zIndex: (theme) => theme.zIndex.modal }}
          PaperProps={{
            sx: {
              width: { xs: '100vw', sm: 420 },
              maxWidth: '100vw',
            },
          }}
        >
          <Box
            sx={{
              display: 'flex',
              flexDirection: 'column',
              height: '100%',
            }}
          >
            <Box
              sx={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                px: 3,
                py: 2,
                borderBottom: (theme) => `1px solid ${theme.palette.divider}`,
              }}
            >
              <Box>
                <Typography variant="h6">
                  {createFromImport ? 'Import database' : 'Create database'}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Select a provider to continue
                </Typography>
              </Box>
              <IconButton aria-label="Close" onClick={closeMenu} size="small">
                <CloseIcon fontSize="small" />
              </IconButton>
            </Box>

            {createFromImport && (
              <Box
                sx={{
                  px: 3,
                  py: 1.5,
                  bgcolor: (theme) => theme.palette.action.hover,
                  borderBottom: (theme) => `1px solid ${theme.palette.divider}`,
                }}
              >
                <Typography variant="caption" color="text.secondary">
                  {techPreviewText}
                </Typography>
              </Box>
            )}

            <Box
              sx={{
                flexGrow: 1,
                overflowY: 'auto',
                p: 2,
                display: 'flex',
                flexDirection: 'column',
                gap: 1,
              }}
            >
              {providers.map((item) => {
                const providerName = item?.metadata?.name ?? '';
                const meta = getProviderMeta(providerName);
                const label = meta?.displayName || providerName;
                return (
                  <Box
                    key={providerName}
                    data-testid={`${createFromImport ? 'import' : 'add'}-db-cluster-button-${providerName}`}
                    component={Link}
                    to="/databases/new"
                    state={{
                      selectedDbProvider: item,
                      showImport: createFromImport,
                    }}
                    onClick={closeMenu}
                    sx={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 1.5,
                      px: 2,
                      py: 1.25,
                      borderRadius: 1,
                      border: (theme) => `1px solid ${theme.palette.divider}`,
                      textDecoration: 'none',
                      color: 'inherit',
                      transition: 'border-color 0.15s, background-color 0.15s',
                      '&:hover': {
                        borderColor: (theme) => theme.palette.primary.main,
                        backgroundColor: (theme) => theme.palette.action.hover,
                      },
                    }}
                  >
                    {meta && (
                      <Avatar
                        src={meta.icon}
                        variant="rounded"
                        sx={{
                          width: 36,
                          height: 36,
                          flexShrink: 0,
                          bgcolor: meta.icon
                            ? 'transparent'
                            : (theme) => theme.palette.primary.main,
                          fontSize: '1rem',
                          textTransform: 'uppercase',
                        }}
                      >
                        {label.charAt(0)}
                      </Avatar>
                    )}
                    <Typography
                      variant="body1"
                      sx={{
                        flexGrow: 1,
                        minWidth: 0,
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {label}
                    </Typography>
                  </Box>
                );
              })}
            </Box>
          </Box>
        </Drawer>
      )}
    </Box>
  ) : null;
};

export default CreateDbButton;
