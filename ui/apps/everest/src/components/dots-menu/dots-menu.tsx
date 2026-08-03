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

import { IconButton, Menu, MenuItem } from '@mui/material';
import MoreHorizOutlinedIcon from '@mui/icons-material/MoreHorizOutlined';
import { DotsMenuProps } from './dots-menu.types';
import { useState, MouseEvent, createElement } from 'react';

export const DotsMenu = ({
  menuProps,
  options,
  iconButtonProps,
}: DotsMenuProps) => {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  const handleClick = (event: MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  return (
    <>
      <IconButton
        aria-label="more"
        id="more-button"
        onClick={handleClick}
        size="small"
        {...iconButtonProps}
      >
        <MoreHorizOutlinedIcon />
      </IconButton>
      <Menu
        id="more-horiz-menu"
        anchorEl={anchorEl}
        open={!!anchorEl}
        onClose={handleClose}
        {...menuProps}
        slotProps={{
          list: {
            'aria-labelledby': 'long-button',
          },
        }}
      >
        {options.map((item) => (
          <MenuItem
            key={item.key}
            onClick={() => {
              item.onClick();
              handleClose();
            }}
            sx={{
              gap: 1,
            }}
            disabled={!!item?.disabled}
          >
            {item.icon && createElement(item.icon, { fontSize: 'small' })}
            {item.children}
          </MenuItem>
        ))}
      </Menu>
    </>
  );
};
