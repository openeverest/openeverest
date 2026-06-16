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

import { useContext, useState } from 'react';
import {
  Divider,
  FormControlLabel,
  FormGroup,
  IconButton,
  Menu,
  MenuItem,
  Switch,
  Typography,
} from '@mui/material';
import PersonOutlineIcon from '@mui/icons-material/PersonOutline';
import { ColorModeContext } from '@percona/design';
import { AuthContext } from 'contexts/auth';
import { getAuthToken } from 'api/session-token';
import { jwtDecode } from 'jwt-decode';
import { Skeleton } from '@mui/material';

interface UserToken {
  preferred_username?: string;
  name?: string;
  email?: string;
  sub: string;
}

const AppBarUserIcon = () => {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const { colorMode, toggleColorMode } = useContext(ColorModeContext);
  const { logout } = useContext(AuthContext);

  const handleMenu = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };
  const token = getAuthToken();

  if (!token) {
    return <Skeleton variant="circular" width={40} height={40} />;
  }

  const decoded = jwtDecode(token) as UserToken;

  const preferredUsername = decoded.preferred_username;
  const name = decoded.name;
  const email = decoded.email;
  const sub =
    decoded.sub?.substring(0, decoded.sub.indexOf(':')) || decoded.sub;

  const userToShow = preferredUsername || name || email || sub;

  return (
    <>
      <IconButton
        size="medium"
        aria-label="user"
        aria-controls="menu-user"
        aria-haspopup="true"
        onClick={handleMenu}
        sx={{ color: 'text.secondary' }}
        data-testid="user-appbar-button"
      >
        <PersonOutlineIcon />
      </IconButton>
      <Menu
        id="menu-user"
        anchorEl={anchorEl}
        keepMounted
        open={Boolean(anchorEl)}
        onClose={handleClose}
      >
        <MenuItem
          disableTouchRipple
          sx={{
            pointerEvents: 'none',
            cursor: 'default',
          }}
        >
          <Typography variant="helperText" color="text.secondary">
            {userToShow}
          </Typography>
        </MenuItem>
        <Divider />
        <MenuItem>
          <FormGroup>
            <FormControlLabel
              control={
                <Switch
                  checked={colorMode === 'dark'}
                  onChange={toggleColorMode}
                />
              }
              label="Dark mode"
              labelPlacement="start"
              sx={{
                ml: 0,
              }}
            />
          </FormGroup>
        </MenuItem>
        <MenuItem onClick={logout}>Log out</MenuItem>
      </Menu>
    </>
  );
};

export default AppBarUserIcon;
