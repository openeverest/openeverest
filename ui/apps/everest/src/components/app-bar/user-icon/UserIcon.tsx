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
import { jwtDecode } from 'jwt-decode';

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
  const token = localStorage.getItem('everestToken');
  const decoded = jwtDecode(token!) as UserToken;

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
