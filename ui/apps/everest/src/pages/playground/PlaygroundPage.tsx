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

import { useState, useCallback } from 'react';
import { Box, Typography, Chip } from '@mui/material';
import { CodeMirrorYamlEditor } from 'components/playground/CodeMirrorYamlEditor';
import { PreviewPanel } from 'components/playground/PreviewPanel';
import { PlaygroundLayout } from 'components/playground/PlaygroundLayout';
import { useYamlValidation } from 'hooks/useYamlValidation';

const DEFAULT_YAML = `# OpenEverest Database Cluster Example
apiVersion: everest.percona.com/v1alpha1
kind: DatabaseCluster
metadata:
  name: my-cluster
  namespace: everest
spec:
  engine:
    type: postgresql
    version: "16"
    replicas: 3
  proxy:
    type: pgbouncer
    replicas: 2
  monitoring:
    enabled: true
`;

/**
 * Playground PoC page — demonstrates a CSP-safe YAML editor
 * running inside OpenEverest's strict nonce-based CSP environment.
 *
 * This is an isolated proof-of-concept. It does NOT integrate with
 * UIGenerator, the database wizard, or any persistence layer.
 */
export const PlaygroundPage = () => {
  const [content, setContent] = useState(DEFAULT_YAML);
  const { parsed, error } = useYamlValidation(content);

  const handleChange = useCallback((value: string) => {
    setContent(value);
  }, []);

  return (
    <Box
      data-testid="playground-page"
      sx={{
        display: 'flex',
        flexDirection: 'column',
        height: 'calc(100vh - 64px)', // account for app bar
        p: 3,
        gap: 2,
      }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1.5,
          flexShrink: 0,
        }}
      >
        <Typography variant="h5" component="h1">
          YAML Playground
        </Typography>
        <Chip
          label="PoC"
          size="small"
          color="info"
          variant="outlined"
          data-testid="poc-badge"
        />
        {error ? (
          <Chip
            label="Invalid YAML"
            size="small"
            color="error"
            data-testid="status-invalid"
          />
        ) : content.trim() ? (
          <Chip
            label="Valid YAML"
            size="small"
            color="success"
            data-testid="status-valid"
          />
        ) : null}
      </Box>

      <Box sx={{ flex: 1, minHeight: 0 }}>
        <PlaygroundLayout
          editor={
            <CodeMirrorYamlEditor
              initialContent={DEFAULT_YAML}
              onChange={handleChange}
            />
          }
          preview={<PreviewPanel parsed={parsed} error={error} />}
        />
      </Box>
    </Box>
  );
};
