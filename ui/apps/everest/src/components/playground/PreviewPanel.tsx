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

import { Box, Typography, Alert, useTheme } from '@mui/material';

export interface PreviewPanelProps {
  /** Parsed YAML value (any valid JS value from YAML parse) */
  parsed: unknown;
  /** Validation error message, if any */
  error: string | null;
}

/**
 * Displays a live preview of parsed YAML content as formatted JSON,
 * or shows validation errors when the YAML is malformed.
 */
export const PreviewPanel = ({ parsed, error }: PreviewPanelProps) => {
  const theme = useTheme();

  if (error) {
    return (
      <Box
        data-testid="preview-panel"
        sx={{ p: 2, height: '100%', overflow: 'auto' }}
      >
        <Alert
          severity="error"
          data-testid="preview-error"
          sx={{ whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}
        >
          {error}
        </Alert>
      </Box>
    );
  }

  if (parsed === null || parsed === undefined) {
    return (
      <Box
        data-testid="preview-panel"
        sx={{
          p: 2,
          height: '100%',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Typography
          color="text.secondary"
          data-testid="preview-empty"
        >
          Enter YAML on the left to see the parsed preview
        </Typography>
      </Box>
    );
  }

  return (
    <Box
      data-testid="preview-panel"
      sx={{ p: 2, height: '100%', overflow: 'auto' }}
    >
      <Typography
        variant="subtitle2"
        color="text.secondary"
        sx={{ mb: 1 }}
      >
        Parsed Output (JSON)
      </Typography>
      <Box
        component="pre"
        data-testid="preview-output"
        sx={{
          m: 0,
          p: 2,
          borderRadius: 1,
          backgroundColor: theme.palette.action.hover,
          fontFamily: '"Roboto Mono", monospace',
          fontSize: '0.8125rem',
          lineHeight: 1.6,
          overflow: 'auto',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
        }}
      >
        {JSON.stringify(parsed, null, 2)}
      </Box>
    </Box>
  );
};
