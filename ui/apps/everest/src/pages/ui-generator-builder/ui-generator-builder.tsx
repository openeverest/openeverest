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

import { Box, Paper } from '@mui/material';
import { useState, useEffect, useRef } from 'react';
import { JsonEditorPanel } from './json-editor-panel/json-editor-panel';
import schemaYaml from 'components/ui-generator/ui-generator.mock.yaml?raw';
import { TopologyUISchemas } from '../../components/ui-generator/ui-generator.types';
import { ErrorBoundary } from 'utils/ErrorBoundary';
import { GenericError } from 'pages/generic-error/GenericError';
import { ErrorContextProvider } from 'utils/ErrorBoundaryProvider';
import { DynamicForm } from './dynamic-form-preview/dynamic-form-preview';
import { formatYamlText, yamlToJson } from './utils/yaml-json-converter';

export const UIGeneratorBuilder = () => {
  const defaultYamlText = schemaYaml;
  const [yamlText, setYamlText] = useState(defaultYamlText);
  const [parsedSchema, setParsedSchema] = useState<TopologyUISchemas | null>(
    null
  );
  const [error, setError] = useState<string>('');
  const [leftWidth, setLeftWidth] = useState(25); // percentage
  const [isDragging, setIsDragging] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    validateYaml(defaultYamlText);
  }, []);

  useEffect(() => {
    if (!isDragging) return;

    const handleMouseMove = (e: MouseEvent) => {
      if (!containerRef.current) return;

      const container = containerRef.current;
      const rect = container.getBoundingClientRect();
      const newLeftWidth = ((e.clientX - rect.left) / rect.width) * 100;

      if (newLeftWidth >= 20 && newLeftWidth <= 80) {
        setLeftWidth(newLeftWidth);
      }
    };

    const handleMouseUp = () => {
      setIsDragging(false);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isDragging]);

  const handleYamlChange = (text: string) => {
    setYamlText(text);
    validateYaml(text);
  };

  const validateYaml = (text: string) => {
    try {
      const parsed = yamlToJson(text);
      setParsedSchema(parsed);
      setError('');
    } catch (err) {
      setError(
        err instanceof Error ? `YAML Error: ${err.message}` : 'Invalid YAML'
      );
      setParsedSchema(null);
    }
  };

  const formatYaml = () => {
    try {
      const parsed = yamlToJson(yamlText);
      const formatted = formatYamlText(yamlText);
      setYamlText(formatted);
      setParsedSchema(parsed);
      setError('');
    } catch {
      setError('Invalid YAML format');
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
        <JsonEditorPanel
          yamlText={yamlText}
          error={error}
          onChange={handleYamlChange}
          onFormat={formatYaml}
        />
      </Box>
      <Box
        onMouseDown={() => setIsDragging(true)}
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
          {/*TODO add custom error boundary for FormBuilder*/}
          {parsedSchema && (
            <ErrorContextProvider>
              <ErrorBoundary fallback={<GenericError />}>
                <DynamicForm schema={parsedSchema as TopologyUISchemas} />
              </ErrorBoundary>
            </ErrorContextProvider>
          )}
        </Paper>
      </Box>
    </Box>
  );
};
