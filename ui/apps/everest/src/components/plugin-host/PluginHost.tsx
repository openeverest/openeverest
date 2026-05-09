import { useParams } from 'react-router-dom';
import { usePlugins } from 'contexts/plugins';
import type { RouteExtension } from '@openeverest/plugin-sdk';
import { Box, Typography } from '@mui/material';
import PluginErrorBoundary from './PluginErrorBoundary';

const PluginHost = () => {
  const { pluginName, '*': subPath } = useParams();
  const { plugins, loading } = usePlugins();

  if (loading) {
    return <Typography>Loading plugins...</Typography>;
  }

  const plugin = plugins.find((p) => p.name === pluginName);
  if (!plugin) {
    return (
      <Box>
        <Typography variant="h5">Plugin not found</Typography>
        <Typography color="text.secondary">
          No plugin registered with name &quot;{pluginName}&quot;.
        </Typography>
      </Box>
    );
  }

  // Find the first route extension with a component
  const routeExtension = plugin.extensions.find(
    (ext): ext is RouteExtension => ext.type === 'route'
  );

  if (!routeExtension?.component) {
    return (
      <Box>
        <Typography variant="h5">{plugin.name}</Typography>
        <Typography color="text.secondary">
          This plugin does not provide a UI component for this route.
        </Typography>
      </Box>
    );
  }

  const PluginComponent = routeExtension.component;
  return (
    <PluginErrorBoundary pluginName={pluginName ?? ''}>
      <PluginComponent pluginName={pluginName ?? ''} subPath={subPath} />
    </PluginErrorBoundary>
  );
};

export default PluginHost;
