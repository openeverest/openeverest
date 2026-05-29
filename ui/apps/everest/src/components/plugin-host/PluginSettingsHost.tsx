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
import { useParams } from 'react-router-dom';
import { Box, Typography } from '@mui/material';
import { usePlugins } from 'contexts/plugins';
import type { SettingsPanelExtension } from '@openeverest/plugin-sdk';
import PluginErrorBoundary from './PluginErrorBoundary';

/**
 * Renders a plugin-contributed settingsPanel matched by the current route
 * segment (:tabs). Placed as a catch-all child of the settings route.
 */
const PluginSettingsHost = () => {
  const { tabs: tabPath = '' } = useParams();
  const { plugins } = usePlugins();

  const match = useMemo(() => {
    for (const plugin of plugins) {
      for (const ext of plugin.extensions) {
        if (
          ext.type === 'settingsPanel' &&
          (ext as SettingsPanelExtension).path === tabPath
        ) {
          return {
            pluginName: plugin.name,
            ext: ext as SettingsPanelExtension,
          };
        }
      }
    }
    return null;
  }, [plugins, tabPath]);

  if (!match) {
    return (
      <Box sx={{ p: 2 }}>
        <Typography color="text.secondary">Unknown settings tab.</Typography>
      </Box>
    );
  }

  const Component = match.ext.component;
  // TODO: pass actual currentUser from auth context when available.
  return (
    <PluginErrorBoundary pluginName={match.pluginName}>
      <Component currentUser="" />
    </PluginErrorBoundary>
  );
};

export default PluginSettingsHost;
