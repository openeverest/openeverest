import { Component, ErrorInfo, ReactNode } from 'react';
import { useParams } from 'react-router-dom';
import { usePlugins } from 'contexts/plugins';
import type { RouteExtension } from '@openeverest/plugin-sdk';
import { Box, Typography, Button } from '@mui/material';

// Error boundary catches runtime errors in plugin components.
interface ErrorBoundaryProps {
  pluginName: string;
  children: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

class PluginErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error(`[plugins] Plugin "${this.props.pluginName}" crashed:`, error, info);
  }

  render() {
    if (this.state.hasError) {
      return (
        <Box sx={{ p: 3 }}>
          <Typography variant="h5" color="error" gutterBottom>
            Plugin Error
          </Typography>
          <Typography color="text.secondary" gutterBottom>
            The plugin &quot;{this.props.pluginName}&quot; encountered an error and could not render.
          </Typography>
          <Typography variant="body2" sx={{ fontFamily: 'monospace', mt: 1, mb: 2 }}>
            {this.state.error?.message}
          </Typography>
          <Button
            variant="outlined"
            onClick={() => this.setState({ hasError: false, error: null })}
          >
            Try Again
          </Button>
        </Box>
      );
    }
    return this.props.children;
  }
}

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
