import { useContext, useState } from 'react';
import {
  Divider,
  IconButton,
  Link,
  Menu,
  MenuItem,
  Typography,
} from '@mui/material';
import HelpIcon from '@mui/icons-material/Help';
import { UpgradeEverestContext } from 'contexts/upgrade-everest';

const AppBarHelpIcon = () => {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const { currentVersion } = useContext(UpgradeEverestContext);

  const handleMenu = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  return (
    <>
      <IconButton
        size="medium"
        aria-label="help"
        aria-controls="menu-help"
        aria-haspopup="true"
        onClick={handleMenu}
        sx={{ color: 'text.secondary' }}
        data-testid="help-appbar-button"
      >
        <HelpIcon />
      </IconButton>
      <Menu
        id="menu-help"
        anchorEl={anchorEl}
        keepMounted
        open={Boolean(anchorEl)}
        onClose={handleClose}
      >
        {/* For this particular menu title, we allow text selection, as it contains the version number */}
        <MenuItem
          disableTouchRipple
          sx={{ cursor: 'text', userSelect: 'text' }}
        >
          <Typography variant="helperText" color="text.secondary">
            {`OpenEverest ${currentVersion}`}
          </Typography>
        </MenuItem>
        <Divider />
        <Link
          underline="none"
          color="inherit"
          target="_blank"
          rel="noopener"
          href="https://openeverest.io/support/"
        >
          <MenuItem onClick={handleClose}>
            <Typography variant="menuText">Get Support</Typography>
          </MenuItem>
        </Link>
        <Link
          underline="none"
          color="inherit"
          target="_blank"
          rel="noopener"
          href="https://openeverest.io/docs/"
        >
          <MenuItem onClick={handleClose}>
            <Typography variant="menuText">Documentation</Typography>
          </MenuItem>
        </Link>
        <Link
          underline="none"
          color="inherit"
          target="_blank"
          rel="noopener"
          href="https://github.com/openeverest/openeverest/issues"
        >
          <MenuItem onClick={handleClose}>
            <Typography variant="menuText">Report issue</Typography>
          </MenuItem>
        </Link>
        <Link
          underline="none"
          color="inherit"
          target="_blank"
          rel="noopener"
          href="https://openeverest.io/#community"
        >
          <MenuItem onClick={handleClose}>
            <Typography variant="menuText">Join Community</Typography>
          </MenuItem>
        </Link>
      </Menu>
    </>
  );
};

export default AppBarHelpIcon;
