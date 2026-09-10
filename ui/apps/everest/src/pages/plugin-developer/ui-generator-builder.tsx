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

import { Box, MenuItem, Paper, TextField } from '@mui/material';
import { useState } from 'react';
import { YamlEditorPanel } from './yaml-editor-panel/yaml-editor-panel';
import schemaYaml from 'components/ui-generator/ui-generator.mock.yaml?raw';
import { TopologyUISchemas } from 'components/ui-generator/ui-generator.types';
import { ErrorBoundary } from 'utils/ErrorBoundary';
import { GenericError } from 'pages/generic-error/GenericError';
import { ErrorContextProvider } from 'utils/ErrorBoundaryProvider';
import { DynamicForm } from './dynamic-form-preview/dynamic-form-preview';
import { formatYamlText } from './utils/yaml-json-converter';
import { useSchemaValidation } from './hooks/use-schema-validation';
import { useSplitPane } from './hooks/use-split-pane';
import { usePreviewNamespace } from './hooks/use-preview-namespace';

export const UIGeneratorBuilder = () => {
  const [yamlText, setYamlText] = useState(schemaYaml);
  const { diagnostics, parsed: parsedSchema } = useSchemaValidation(yamlText);
  const { containerRef, leftWidth, isDragging, startDragging } = useSplitPane();
  const {
    namespaces,
    selectedNamespace,
    setSelectedNamespace,
    isLoading: namespacesLoading,
  } = usePreviewNamespace();

  const formatYaml = () => {
    try {
      setYamlText(formatYamlText(yamlText));
    } catch {
      // Unparseable YAML can't be formatted; the syntax error is already shown.
    }
  };

  return (
    <Box
      ref={containerRef}
      sx={{
        height: 'calc(100vh - 150px)',
        display: 'flex',
        flexDirection: 'row',
        overflow: 'hidden',
      }}
    >
      <Box
        sx={[
          {
            width: `${leftWidth}%`,
            height: '100%',
            display: 'flex',
            flexDirection: 'column',
          },
          isDragging
            ? {
                transition: 'none',
              }
            : {
                transition: 'width 0.2s ease',
              },
        ]}
      >
        <YamlEditorPanel
          yamlText={yamlText}
          diagnostics={diagnostics}
          onChange={setYamlText}
          onFormat={formatYaml}
        />
      </Box>
      <Box
        onMouseDown={startDragging}
        sx={[
          {
            width: '8px',
            height: '100%',
            backgroundColor: 'divider',
            cursor: 'col-resize',
            userSelect: 'none',
            '&:hover': {
              backgroundColor: 'primary.main',
              opacity: 0.6,
            },
          },
          isDragging
            ? {
                transition: 'none',
              }
            : {
                transition: 'backgroundColor 0.2s ease',
              },
        ]}
      />
      <Box
        sx={[
          {
            flex: 1,
            width: `${100 - leftWidth}%`,
            height: '100%',
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
          },
          isDragging
            ? {
                transition: 'none',
              }
            : {
                transition: 'width 0.2s ease',
              },
        ]}
      >
        <Paper
          elevation={0}
          sx={{
            p: 2,
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: 1,
            width: '100%',
            height: '100%',
            overflowY: 'auto',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <TextField
            select
            size="small"
            label="Preview namespace"
            value={selectedNamespace}
            onChange={(e) => setSelectedNamespace(e.target.value)}
            disabled={namespacesLoading || namespaces.length === 0}
            helperText="Namespace used to load data-source providers (e.g. storage classes, monitoring configs)"
            sx={{ mb: 2, maxWidth: 360 }}
          >
            {namespaces.map((ns) => (
              <MenuItem key={ns} value={ns}>
                {ns}
              </MenuItem>
            ))}
          </TextField>
          {/*TODO add custom error boundary for FormBuilder*/}
          {parsedSchema && (
            <ErrorContextProvider>
              <ErrorBoundary fallback={<GenericError />}>
                <DynamicForm
                  schema={parsedSchema as TopologyUISchemas}
                  namespace={selectedNamespace}
                />
              </ErrorBoundary>
            </ErrorContextProvider>
          )}
        </Paper>
      </Box>
    </Box>
  );
};
