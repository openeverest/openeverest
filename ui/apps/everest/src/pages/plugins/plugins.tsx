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
  Card,
  CardActionArea,
  CardContent,
  Chip,
  CircularProgress,
  Grid,
  Typography,
} from '@mui/material';
import ExtensionIcon from '@mui/icons-material/Extension';
import { useNavigate } from 'react-router-dom';
import { useAllPlugins } from 'hooks/api/plugins/useAllPlugins';
import type { PluginDescriptor } from 'hooks/api/plugins/usePluginsForNamespace';

const extensionTypeLabels: Record<string, string> = {
  route: 'Page',
  sidebarItem: 'Sidebar',
  clusterDetailTab: 'Instance Tab',
  clusterAction: 'Instance Action',
  clusterCard: 'Instance Card',
  globalDashboardWidget: 'Dashboard Widget',
  settingsPanel: 'Settings Tab',
  instanceCreateFormSection: 'Create Form Section',
  instanceEditFormSection: 'Edit Form Section',
};

function ExtensionChips({ plugin }: { plugin: PluginDescriptor }) {
  if (!plugin.extensionPoints?.length) return null;

  const types = [...new Set(plugin.extensionPoints.map((ep) => ep.type))];

  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 1 }}>
      {types.map((type) => (
        <Chip
          key={type}
          label={extensionTypeLabels[type] || type}
          size="small"
          variant="outlined"
        />
      ))}
    </Box>
  );
}

export const PluginsPage = () => {
  const { data: plugins, isLoading, error } = useAllPlugins();
  const navigate = useNavigate();

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h5" sx={{ mb: 3 }}>
        Plugins
      </Typography>

      {isLoading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 6 }}>
          <CircularProgress />
        </Box>
      )}

      {error && <Typography color="error">Failed to load plugins.</Typography>}

      {!isLoading && plugins?.length === 0 && (
        <Box sx={{ textAlign: 'center', mt: 6 }}>
          <ExtensionIcon sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
          <Typography variant="h6" color="text.secondary">
            No plugins installed
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Plugins extend OpenEverest with additional features.
          </Typography>
        </Box>
      )}

      {plugins && plugins.length > 0 && (
        <Grid container spacing={2}>
          {plugins.map((plugin) => (
            <Grid item xs={12} sm={6} md={4} key={plugin.name}>
              <Card variant="outlined" sx={{ height: '100%' }}>
                <CardActionArea
                  onClick={() => navigate(`/plugins/${plugin.name}`)}
                  sx={{ height: '100%' }}
                >
                  <CardContent>
                    <Box
                      sx={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 1,
                        mb: 1,
                      }}
                    >
                      {plugin.icon ? (
                        <Box
                          component="img"
                          src={plugin.icon}
                          alt=""
                          sx={{ width: 32, height: 32, borderRadius: 1 }}
                        />
                      ) : (
                        <ExtensionIcon
                          sx={{ fontSize: 32, color: 'primary.main' }}
                        />
                      )}
                      <Box sx={{ minWidth: 0 }}>
                        <Typography variant="subtitle1" noWrap>
                          {plugin.displayName}
                        </Typography>
                        {plugin.vendor && (
                          <Typography variant="caption" color="text.secondary">
                            by {plugin.vendor}
                          </Typography>
                        )}
                      </Box>
                      {plugin.version && (
                        <Chip
                          label={`v${plugin.version}`}
                          size="small"
                          color="primary"
                          variant="outlined"
                          sx={{ ml: 'auto', flexShrink: 0 }}
                        />
                      )}
                    </Box>

                    {plugin.description && (
                      <Typography
                        variant="body2"
                        color="text.secondary"
                        sx={{
                          mt: 1,
                          display: '-webkit-box',
                          WebkitLineClamp: 3,
                          WebkitBoxOrient: 'vertical',
                          overflow: 'hidden',
                        }}
                      >
                        {plugin.description}
                      </Typography>
                    )}

                    <ExtensionChips plugin={plugin} />
                  </CardContent>
                </CardActionArea>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}
    </Box>
  );
};
