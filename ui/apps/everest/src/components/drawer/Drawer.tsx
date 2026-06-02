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

import { useContext, useMemo } from 'react';
import {
  IconButton,
  List,
  Drawer as MuiDrawer,
  Toolbar,
  styled,
} from '@mui/material';
import KeyboardDoubleArrowRightIcon from '@mui/icons-material/KeyboardDoubleArrowRight';
import KeyboardDoubleArrowLeftIcon from '@mui/icons-material/KeyboardDoubleArrowLeft';
import ExtensionIcon from '@mui/icons-material/Extension';
import { DRAWER_WIDTH, ROUTES } from './Drawer.constants';
import { closedMixin, openedMixin } from './Drawer.utils';
import { NavItem } from '../nav-item/NavItem';
import { DrawerContext } from 'contexts/drawer/drawer.context';
import { usePlugins } from 'contexts/plugins';
import { EverestRoute } from './Drawer.types';

const DrawerHeader = styled('div')(({ theme }) => ({
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'flex-end',
  padding: theme.spacing(0, 1),
  // necessary for content to be below app bar
  ...theme.mixins.toolbar,
}));

const StyledDrawer = styled(MuiDrawer, {
  shouldForwardProp: (prop) => prop !== 'open',
})(({ theme, open }) => ({
  width: DRAWER_WIDTH,
  flexShrink: 0,
  whiteSpace: 'nowrap',
  boxSizing: 'border-box',
  ...(open && {
    ...openedMixin(theme),
    '& .MuiDrawer-paper': openedMixin(theme),
  }),
  ...(!open && {
    ...closedMixin(theme),
    '& .MuiDrawer-paper': closedMixin(theme),
  }),
}));

const DrawerContent = ({ open }: { open: boolean }) => {
  const { toggleOpen, setOpen, activeBreakpoint } = useContext(DrawerContext);
  const { plugins } = usePlugins();

  const allRoutes: EverestRoute[] = useMemo(() => {
    const pluginRoutes: EverestRoute[] = plugins.flatMap((plugin) =>
      plugin.extensions
        .filter((ext) => ext.type === 'sidebarItem')
        .map((ext) => ({
          to: `/plugins/${plugin.name}`,
          text: ext.label,
          icon: ext.icon || ExtensionIcon,
        }))
    );
    return [...ROUTES, ...pluginRoutes];
  }, [plugins]);

  return (
    <>
      <DrawerHeader data-testid={`${activeBreakpoint}-drawer-header`}>
        <IconButton
          aria-label="open drawer"
          data-testid="open-drawer-button"
          edge="start"
          onClick={toggleOpen}
        >
          {open ? (
            <KeyboardDoubleArrowLeftIcon />
          ) : (
            <KeyboardDoubleArrowRightIcon />
          )}
        </IconButton>
      </DrawerHeader>
      <List>
        {allRoutes.map(({ to, icon, text }) => (
          <NavItem
            onClick={() => setOpen(false)}
            to={to}
            open={open}
            icon={icon}
            text={text}
            key={to}
          />
        ))}
      </List>
    </>
  );
};

const TabletDrawer = () => {
  const { open } = useContext(DrawerContext);

  return (
    <>
      <StyledDrawer variant="permanent" open={false}>
        <Toolbar />
        <DrawerContent open={false} />
      </StyledDrawer>
      <MuiDrawer
        anchor="left"
        variant="temporary"
        open={open}
        sx={{ '& .MuiDrawer-paper': { width: DRAWER_WIDTH } }}
      >
        <Toolbar />
        <DrawerContent open={open} />
      </MuiDrawer>
    </>
  );
};

const DesktopDrawer = () => {
  const { open } = useContext(DrawerContext);

  return (
    <StyledDrawer variant="permanent" open={open}>
      <Toolbar />
      <DrawerContent open={open} />
    </StyledDrawer>
  );
};

const MobileDrawer = () => {
  const { open } = useContext(DrawerContext);

  return (
    <MuiDrawer
      anchor="left"
      variant="temporary"
      open={open}
      ModalProps={{
        keepMounted: true, // Better open performance on mobile.
      }}
      sx={{ '& .MuiDrawer-paper': { width: DRAWER_WIDTH } }}
    >
      <DrawerContent open={open} />
    </MuiDrawer>
  );
};

export const Drawer = () => {
  const { activeBreakpoint } = useContext(DrawerContext);

  if (activeBreakpoint === 'mobile') {
    return <MobileDrawer />;
  }

  if (activeBreakpoint === 'desktop') {
    return <DesktopDrawer />;
  }

  return <TabletDrawer />;
};
