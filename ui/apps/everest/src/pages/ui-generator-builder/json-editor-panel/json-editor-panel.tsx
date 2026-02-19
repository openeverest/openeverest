// everest
// Copyright (C) 2023 Percona LLC
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

import { Typography, Paper, Button, Stack } from '@mui/material';
import Editor from '@monaco-editor/react';

interface JsonEditorPanelProps {
  jsonText: string;
  error: string;
  onChange: (value: string) => void;
  onFormat: () => void;
}

export const JsonEditorPanel = ({
  jsonText,
  error,
  onChange,
  onFormat,
}: JsonEditorPanelProps) => {
  //TODO change theme by user preference
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
        backgroundColor: '#f6f8fa',
      }}
    >
      <Typography variant="h6" sx={{ mb: 2 }}>
        JSON Editor
      </Typography>
      <Editor
        height="100%"
        defaultLanguage="json"
        value={jsonText}
        onChange={(value) => onChange(value || '')}
        theme="light"
        options={{
          minimap: { enabled: false },
          fontSize: 13,
          wordWrap: 'off',
          scrollBeyondLastLine: false,
          formatOnPaste: true,
          formatOnType: true,
          folding: true,
          foldingHighlight: true,
        }}
      />
      {error && (
        <Typography
          variant="caption"
          color="error"
          sx={{ mt: 1, whiteSpace: 'pre-wrap' }}
        >
          {error}
        </Typography>
      )}
      <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
        <Button size="small" variant="contained" onClick={onFormat}>
          Format JSON
        </Button>
      </Stack>
    </Paper>
  );
};
