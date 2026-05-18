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

import { Box, useTheme } from '@mui/material';
import { useYamlEditor } from 'hooks/useYamlEditor';

export interface CodeMirrorYamlEditorProps {
  /** Initial YAML content to populate the editor */
  initialContent?: string;
  /** Called on every document change with the full editor content */
  onChange?: (value: string) => void;
}

/**
 * A CSP-safe YAML editor powered by CodeMirror 6.
 *
 * This component is the architectural core of the playground PoC.
 * It demonstrates that CodeMirror 6 can function under OpenEverest's
 * strict nonce-based CSP without any policy weakening.
 */
export const CodeMirrorYamlEditor = ({
  initialContent,
  onChange,
}: CodeMirrorYamlEditorProps) => {
  const theme = useTheme();
  const { containerRef } = useYamlEditor({ initialContent, onChange });

  return (
    <Box
      ref={containerRef}
      role="textbox"
      aria-label="YAML editor"
      aria-multiline="true"
      data-testid="yaml-editor"
      sx={{
        flex: 1,
        overflow: 'auto',
        border: `1px solid ${theme.palette.divider}`,
        borderRadius: 1,
        '& .cm-editor': {
          height: '100%',
          outline: 'none',
        },
        '& .cm-editor.cm-focused': {
          outline: `2px solid ${theme.palette.primary.main}`,
          outlineOffset: '-1px',
        },
        '& .cm-scroller': {
          fontFamily: '"Roboto Mono", monospace',
          fontSize: '0.875rem',
          lineHeight: 1.6,
        },
        '& .cm-gutters': {
          backgroundColor: theme.palette.action.hover,
          borderRight: `1px solid ${theme.palette.divider}`,
        },
        '& .cm-content': {
          padding: theme.spacing(1),
        },
      }}
    />
  );
};
