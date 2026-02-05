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

import { useEffect, useState } from 'react';
import {
  Box,
  Button,
  Divider,
  Menu,
  MenuItem,
  Skeleton,
  Tooltip,
  Typography,
} from '@mui/material';
import { ArrowDropDownIcon } from '@mui/x-date-pickers/icons';
import { dbEngineToDbType } from '@percona/utils';
import { Link, useNavigate } from 'react-router-dom';
import { useDBEnginesForDbEngineTypes } from 'hooks';
import { useNamespacePermissionsForResource } from 'hooks/rbac';
import { humanizeDbType } from 'utils/db';
import { useDataImporters } from 'hooks/api/data-importers/useDataImporters';

export const CreateDbButton = ({
  createFromImport = false,
}: {
  createFromImport?: boolean;
}) => {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [showDropdownButton, setShowDropdownButton] = useState(false);
  const { canCreate } = useNamespacePermissionsForResource('database-clusters');

  const { data: availableDbImporters } = useDataImporters();
  const supportedEngineTypesForImport = new Set(
    availableDbImporters?.items
      .map((importer) => importer.spec.supportedEngines)
      .flat()
  );

  const open = Boolean(anchorEl);

  const [allAvailableDbTypes, availableDbTypesFetching] =
    useDBEnginesForDbEngineTypes(undefined, {
      refetchInterval: 30 * 1000,
    });

  const availableDbTypes = allAvailableDbTypes.filter((item) =>
    item.dbEngines.some((engine) =>
      createFromImport
        ? supportedEngineTypesForImport.has(engine.dbEngine!.type)
        : true
    )
  );

  const availableEngines = availableDbTypes.filter(
    (item) =>
      !!item.available &&
      item.dbEngines.some((engine) => canCreate.includes(engine.namespace))
  );
  const navigate = useNavigate();

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    if (availableEngines.length > 1) {
      event.stopPropagation();
      setAnchorEl(event.currentTarget);
    } else {
      navigate('/databases/new', {
        state: {
          selectedDbEngine: availableEngines[0].type,
          showImport: createFromImport,
        },
      });
    }
  };
  const closeMenu = () => {
    setAnchorEl(null);
  };

  useEffect(() => {
    if (availableDbTypesFetching) {
      setShowDropdownButton(false);
    } else {
      setTimeout(() => {
        setShowDropdownButton(true);
      }, 300);
    }
  }, [availableDbTypesFetching]);

  const buttonStyle = { display: 'flex', minHeight: '34px', width: '165px' };
  const skeletonStyle = {
    ...buttonStyle,
    borderRadius: '128px',
  };

  const showTechPreviewTooltip =
    createFromImport && availableEngines.length === 1;

  const techPreviewText = 'Technical Preview';

  const createButton = (
    <Button
      data-testid={`${createFromImport ? 'import' : 'add'}-db-cluster-button`}
      size="small"
      variant={createFromImport ? 'text' : 'contained'}
      sx={buttonStyle}
      aria-controls={
        open
          ? `${createFromImport ? 'import' : 'add'}
            -db-cluster-button-menu`
          : undefined
      }
      aria-haspopup="true"
      aria-expanded={open ? 'true' : undefined}
      onClick={handleClick}
      endIcon={availableEngines.length > 1 && <ArrowDropDownIcon />}
    >
      {createFromImport ? 'Import' : 'Create database'}
    </Button>
  );

  return availableEngines.length > 0 ? (
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
      {availableEngines.length > 1 && (
        <Menu
          data-testid={`${
            createFromImport ? 'import' : 'add'
          }-db-cluster-button-menu`}
          anchorEl={anchorEl}
          open={open}
          onClose={closeMenu}
          MenuListProps={{
            'aria-labelledby': 'basic-button',
            sx: { width: anchorEl && anchorEl.offsetWidth },
          }}
        >
          {
            <Box>
              {createFromImport && (
                <>
                  <MenuItem
                    disableTouchRipple
                    sx={{
                      pointerEvents: 'none',
                      cursor: 'default',
                    }}
                  >
                    <Typography
                      sx={{ fontSize: '14px !important' }}
                      color="text.secondary"
                    >
                      {techPreviewText}
                    </Typography>
                  </MenuItem>
                  <Divider />
                </>
              )}
              {availableDbTypes.map((item) => (
                <MenuItem
                  data-testid={`${createFromImport ? 'import' : 'add'}-db-cluster-button-${item.type}`}
                  disabled={!item.available}
                  key={item.type}
                  component={Link}
                  to="/databases/new"
                  sx={{
                    display: 'flex',
                    gap: 1,
                    alignItems: 'center',
                    px: 2,
                    py: '10px',
                  }}
                  state={{
                    selectedDbEngine: item.type,
                    showImport: createFromImport,
                  }}
                >
                  {humanizeDbType(dbEngineToDbType(item.type))}
                </MenuItem>
              ))}
            </Box>
          }
        </Menu>
      )}
    </Box>
  ) : null;
};

export default CreateDbButton;
