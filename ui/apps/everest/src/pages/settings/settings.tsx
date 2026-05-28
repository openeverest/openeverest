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

import { useMemo } from 'react';
import { Box, Tab, Tabs } from '@mui/material';
import { Link, Navigate, Outlet, useMatch } from 'react-router-dom';

import { Messages } from './settings.messages';
import { SettingsTabs } from './settings.types';
import { usePlugins } from 'contexts/plugins';
import type { SettingsPanelExtension } from '@openeverest/plugin-sdk';

export const Settings = () => {
  const routeMatch = useMatch('/settings/:tabs/:detail?');
  const currentTab = routeMatch?.params?.tabs;
  const { plugins } = usePlugins();

  const pluginSettingsTabs = useMemo(
    () =>
      plugins.flatMap((p) =>
        p.extensions
          .filter((ext): ext is SettingsPanelExtension => ext.type === 'settingsPanel')
          .map((ext) => ({ pluginName: p.name, ...ext })),
      ),
    [plugins],
  );

  if (!currentTab) {
    return <Navigate to={SettingsTabs.storageLocations} replace />;
  }

  return (
    <Box sx={{ width: '100%' }}>
      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs
          value={currentTab}
          variant="scrollable"
          allowScrollButtonsMobile
          aria-label="nav tabs"
        >
          {(Object.keys(SettingsTabs) as Array<keyof typeof SettingsTabs>).map(
            (item) => (
              <Tab
                label={Messages.tabs[item]}
                key={SettingsTabs[item]}
                value={SettingsTabs[item]}
                to={SettingsTabs[item]}
                component={Link}
              />
            )
          )}
          {pluginSettingsTabs.map((pt) => (
            <Tab
              label={pt.label}
              key={`plugin-${pt.pluginName}-${pt.path}`}
              value={pt.path}
              to={pt.path}
              component={Link}
              data-testid={`plugin-settings-tab-${pt.path}`}
            />
          ))}
        </Tabs>
      </Box>
      <Outlet />
    </Box>
  );
};
