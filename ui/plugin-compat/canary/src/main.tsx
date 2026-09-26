// Probe plugin for the host/plugin MUI independence suite: exercises the pieces
// most likely to break across MUI versions (theme inheritance, portals, modals,
// hooks through the host React) and reports what it runs on.
import { useState } from 'react';
import * as React from 'react';
import { Button as DirectMuiButton, version as muiVersion } from '@mui/material';
import type { PluginApi, PluginRegisterFn } from '@openeverest/plugin-sdk';
import {
  PluginThemeProvider,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Menu,
  MenuItem,
  Paper,
  TextField,
  Tooltip,
  Typography,
} from '@openeverest/ui-lib';

interface CanaryProbe {
  muiVersion: string;
  reactIsHost: boolean;
}

declare global {
  interface Window {
    __pluginProbe__?: Record<string, CanaryProbe>;
  }
}

let cssNonce: string | undefined;

const CanaryPage = () => {
  const [count, setCount] = useState(0);
  const [text, setText] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null);

  return (
    <PluginThemeProvider cacheKey="canary" nonce={cssNonce}>
      <Box data-testid="canary-root" sx={{ p: 3, display: 'flex', flexDirection: 'column', gap: 2 }}>
        <Typography variant="h5" data-testid="canary-heading">
          Canary
        </Typography>
        <Typography variant="body1" data-testid="canary-body">
          Plugin running MUI {muiVersion}
        </Typography>
        <Paper data-testid="canary-paper" sx={{ p: 2 }}>
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
            <Button variant="contained" data-testid="canary-button">
              Primary
            </Button>
            {/* Plugins import MUI directly for anything ui-lib doesn't re-export. */}
            <DirectMuiButton variant="contained" data-testid="canary-direct-button">
              Direct MUI
            </DirectMuiButton>
            <Button
              variant="outlined"
              data-testid="canary-counter"
              onClick={() => setCount((c) => c + 1)}
            >
              Clicked {count}
            </Button>
            <Chip label="success" color="success" data-testid="canary-chip" />
            <Tooltip title="Canary tooltip">
              <Button data-testid="canary-tooltip-anchor">Hover me</Button>
            </Tooltip>
          </Box>
        </Paper>
        <TextField
          label="Echo"
          size="small"
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
        <Typography data-testid="canary-echo">{text}</Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button data-testid="canary-dialog-open" onClick={() => setDialogOpen(true)}>
            Open dialog
          </Button>
          <Button data-testid="canary-menu-open" onClick={(e) => setMenuAnchor(e.currentTarget)}>
            Open menu
          </Button>
        </Box>
        <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)}>
          <DialogTitle>Canary dialog</DialogTitle>
          <DialogContent>Rendered in a portal outside the plugin root.</DialogContent>
          <DialogActions>
            <Button data-testid="canary-dialog-close" onClick={() => setDialogOpen(false)}>
              Close
            </Button>
          </DialogActions>
        </Dialog>
        <Menu anchorEl={menuAnchor} open={!!menuAnchor} onClose={() => setMenuAnchor(null)}>
          <MenuItem data-testid="canary-menu-item" onClick={() => setMenuAnchor(null)}>
            Menu item
          </MenuItem>
        </Menu>
      </Box>
    </PluginThemeProvider>
  );
};

const register: PluginRegisterFn = (api: PluginApi) => {
  cssNonce = api.cssNonce;
  window.__pluginProbe__ = {
    ...window.__pluginProbe__,
    canary: {
      muiVersion: muiVersion ?? 'unknown',
      reactIsHost: React.useState === api.React.useState,
    },
  };
  api.registerExtension({ type: 'sidebarItem', label: 'Canary' });
  api.registerExtension({ type: 'route', label: 'Canary', component: CanaryPage });
};

export default register;
