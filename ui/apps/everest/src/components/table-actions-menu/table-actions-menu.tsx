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

import { useState } from 'react';
import { Box, IconButton, Menu } from '@mui/material';
import MoreHorizIcon from '@mui/icons-material/MoreHoriz';
import { TableActionsMenuProps } from './table-actions-menu.types';
import { MoreVert } from '@mui/icons-material';

export const TableActionsMenu = ({
  buttonProps,
  menuItems,
  menuProps,
  isVertical = false,
  buttonColor,
}: TableActionsMenuProps) => {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const open = Boolean(anchorEl);
  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    setAnchorEl(event.currentTarget);
    if (buttonProps?.onClick) {
      buttonProps?.onClick(event);
    }
  };
  const handleClose = (event: React.MouseEvent) => {
    setAnchorEl(null);
    event.stopPropagation();
  };

  const showMenu = menuProps?.showMenu
    ? menuProps?.showMenu
    : menuItems?.length > 0;

  return (
    <Box>
      {showMenu && (
        <>
          <IconButton
            data-testid="row-actions-menu-button"
            aria-haspopup="true"
            aria-controls={open ? 'basic-menu' : undefined}
            aria-expanded={open ? 'true' : undefined}
            onClick={handleClick}
            {...buttonProps}
            sx={{ color: buttonColor }}
          >
            {isVertical ? <MoreVert /> : <MoreHorizIcon />}
          </IconButton>
          <Menu
            data-testid="row-actions-menu"
            open={open}
            onClose={(
              event: React.MouseEvent<HTMLLIElement, MouseEvent>,
              reason
            ) => {
              if (menuProps?.onClose) {
                menuProps?.onClose(event, reason);
              }
              handleClose(event);
            }}
            anchorEl={anchorEl}
            onClick={(event) => {
              handleClose(event);
            }}
            {...menuProps}
            slotProps={{
              list: {
                'aria-labelledby': 'row-actions-button',
              },
            }}
          >
            {...menuItems}
          </Menu>
        </>
      )}
    </Box>
  );
};

export default TableActionsMenu;
