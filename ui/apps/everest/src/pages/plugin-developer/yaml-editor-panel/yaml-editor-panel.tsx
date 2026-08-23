// everest
// Copyright (C) 2023 Percona LLC
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

import { Typography, Paper, Button, Stack, Box, useTheme } from '@mui/material';
import { CodeMirrorEditor } from '../editor/CodeMirrorEditor';
import { Diagnostic } from '../editor/types';

interface YamlEditorPanelProps {
  yamlText: string;
  diagnostics: Diagnostic[];
  onChange: (value: string) => void;
  onFormat: () => void;
}

export const YamlEditorPanel = ({
  yamlText,
  diagnostics,
  onChange,
  onFormat,
}: YamlEditorPanelProps) => {
  const theme = useTheme();
  const errorCount = diagnostics.filter((d) => d.severity === 'error').length;

  return (
    <Paper
      elevation={0}
      sx={{
        p: 2,
        border: '1px solid',
        borderColor: 'divider',
        borderRadius: 1,
        width: '100%',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <Typography variant="h6" sx={{ mb: 2 }}>
        YAML Editor
      </Typography>
      <Box sx={{ flex: 1, minHeight: 0 }}>
        <CodeMirrorEditor
          value={yamlText}
          onChange={onChange}
          diagnostics={diagnostics}
          theme={theme.palette.mode === 'dark' ? 'dark' : 'light'}
        />
      </Box>
      {errorCount > 0 && (
        <Typography variant="caption" color="error" sx={{ mt: 1 }}>
          {errorCount} {errorCount === 1 ? 'error' : 'errors'} — fix to preview
        </Typography>
      )}
      <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
        <Button size="small" variant="contained" onClick={onFormat}>
          Format YAML
        </Button>
      </Stack>
    </Paper>
  );
};
