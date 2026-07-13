import { useContext } from 'react';
import { Box, IconButton, AppBar as MuiAppBar, Toolbar } from '@mui/material';
import KeyboardDoubleArrowRightIcon from '@mui/icons-material/KeyboardDoubleArrowRight';
import { EverestHorizontalIcon } from '@percona/ui-lib';
import { Link } from 'react-router-dom';
import { DrawerContext } from 'contexts/drawer/drawer.context';
import AppBarHelpIcon from './help-icon/HelpIcon';
import AppBarUserIcon from './user-icon/UserIcon';

export const AppBar = () => {
  const { activeBreakpoint, toggleOpen } = useContext(DrawerContext);

  return (
    <Box>
      <MuiAppBar
        position="fixed"
        sx={(theme) => ({
          ...(activeBreakpoint !== 'mobile' && {
            zIndex: theme.zIndex.drawer + 1,
          }),
        })}
      >
        <Toolbar>
          {activeBreakpoint === 'mobile' && (
            <IconButton
              color="inherit"
              aria-label="open drawer"
              onClick={toggleOpen}
              edge="start"
              sx={{ mr: 2 }}
            >
              <KeyboardDoubleArrowRightIcon />
            </IconButton>
          )}
          <Link to="/">
            <EverestHorizontalIcon sx={{ height: '32px', width: 'auto' }} />
          </Link>
          <Box sx={{ ml: 'auto' }}>
            <AppBarHelpIcon />
            <AppBarUserIcon />
          </Box>
        </Toolbar>
      </MuiAppBar>
    </Box>
  );
};
