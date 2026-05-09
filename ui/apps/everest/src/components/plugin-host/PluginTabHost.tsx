import { useContext, useMemo } from 'react';
import { useParams } from 'react-router-dom';
import { Box, Typography } from '@mui/material';
import { usePlugins } from 'contexts/plugins';
import type { ClusterDetailTabExtension } from '@openeverest/plugin-sdk';
import { DbInstanceContext } from 'pages/db-cluster-details/dbCluster.context';
import PluginErrorBoundary from './PluginErrorBoundary';

/**
 * Renders a plugin-contributed clusterDetailTab matched by the current route
 * segment (:tabs). Placed as a catch-all child of the instance detail route.
 */
const PluginTabHost = () => {
  const { namespace = '', instanceName = '', tabs: tabPath = '' } = useParams();
  const { plugins } = usePlugins();
  const { instance } = useContext(DbInstanceContext);

  const match = useMemo(() => {
    for (const plugin of plugins) {
      for (const ext of plugin.extensions) {
        if (ext.type === 'clusterDetailTab' && (ext as ClusterDetailTabExtension).path === tabPath) {
          return { pluginName: plugin.name, ext: ext as ClusterDetailTabExtension };
        }
      }
    }
    return null;
  }, [plugins, tabPath]);

  if (!match) {
    return (
      <Box sx={{ p: 2 }}>
        <Typography color="text.secondary">Unknown tab.</Typography>
      </Box>
    );
  }

  const Component = match.ext.component;
  return (
    <PluginErrorBoundary pluginName={match.pluginName}>
      <Component cluster={instance} namespace={namespace} instanceName={instanceName} />
    </PluginErrorBoundary>
  );
};

export default PluginTabHost;
