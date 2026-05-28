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

import {
  Box,
  IconButton,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Tooltip,
} from '@mui/material';
import React from 'react';
import { NavLink, useNavigate } from 'react-router-dom';
import { NavItemProps } from './NavItem.types';

export const NavItem = ({
  open,
  icon,
  text,
  to,
  onClick,
  ...listItemProps
}: NavItemProps) => {
  const navigate = useNavigate();

  const redirect = (redirectUrl: string) => {
    navigate(redirectUrl);
    onClick();
  };

  return (
    <ListItem disablePadding sx={{ display: 'block' }} {...listItemProps}>
      {open ? (
        <ListItemButton
          component={NavLink}
          to={to}
          sx={{
            minHeight: 48,
            justifyContent: 'initial',
            px: 2.5,
          }}
          onClick={onClick}
        >
          <ListItemIcon
            sx={{
              minWidth: 0,
              mr: 3,
              justifyContent: 'center',
            }}
          >
            {typeof icon === 'string' ? (
              <img src={icon} alt="" width={24} height={24} />
            ) : (
              React.createElement(icon)
            )}
          </ListItemIcon>

          <ListItemText primary={text} />
        </ListItemButton>
      ) : (
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            minHeight: 48,
            px: '12px',
          }}
        >
          <Tooltip title={text} placement="right" arrow>
            <IconButton onClick={() => redirect(to)}>
              {typeof icon === 'string' ? (
                <img src={icon} alt="" width={24} height={24} />
              ) : (
                React.createElement(icon)
              )}
            </IconButton>
          </Tooltip>
        </Box>
      )}
    </ListItem>
  );
};
