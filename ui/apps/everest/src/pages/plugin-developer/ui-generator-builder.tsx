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

import { Box, MenuItem, Tab, Tabs, TextField } from '@mui/material';
import { useEffect, useState } from 'react';
import { YamlEditorPanel } from './yaml-editor-panel/yaml-editor-panel';
import schemaYaml from 'components/ui-generator/ui-generator.mock.yaml?raw';
import { ErrorBoundary } from 'utils/ErrorBoundary';
import { GenericError } from 'pages/generic-error/GenericError';
import { ErrorContextProvider } from 'utils/ErrorBoundaryProvider';
import { DynamicForm } from './dynamic-form-preview/dynamic-form-preview';
import { formatYamlText } from './utils/yaml-json-converter';
import { useSchemaValidation } from './hooks/use-schema-validation';
import { useSplitPane } from './hooks/use-split-pane';
import { usePreviewNamespace } from './hooks/use-preview-namespace';
import { useSchemaStorage } from './hooks/use-schema-storage';
import { SchemaToolbar } from './schema-toolbar/schema-toolbar';
import { OutputPanel } from './output-panel/output-panel';
import RoundedBox from 'components/rounded-box';
import { Messages } from './ui-generator-builder.messages';

// One height for both header strips keeps the panel borders below them level.
const HEADER_HEIGHT = 48;

// Panes animate their resize, except while dragging, when the transition would
// lag the pointer.
const paneTransition = (isDragging: boolean, property: string) => ({
  transition: isDragging ? 'none' : `${property} 0.2s ease`,
});

export const UIGeneratorBuilder = () => {
  const {
    initialDraft,
    names,
    saveDraft,
    saveSchema,
    loadSchema,
    deleteSchema,
  } = useSchemaStorage(schemaYaml);
  const [yamlText, setYamlText] = useState(initialDraft);
  const [rightTab, setRightTab] = useState(0);
  const [output, setOutput] = useState<Record<string, unknown> | null>(null);
  // Bumped when a whole schema is swapped in, to remount the preview. Editing
  // the YAML deliberately doesn't bump it, so the form keeps its values.
  const [previewKey, setPreviewKey] = useState(0);
  const { diagnostics, parsed: parsedSchema } = useSchemaValidation(yamlText);
  const { containerRef, leftWidth, isDragging, startDragging } = useSplitPane();
  const {
    namespaces,
    selectedNamespace,
    setSelectedNamespace,
    isLoading: namespacesLoading,
  } = usePreviewNamespace();

  // Autosave the working schema so a reload restores the developer's draft.
  useEffect(() => {
    saveDraft(yamlText);
  }, [yamlText, saveDraft]);

  // Editing the YAML invalidates a payload built from the previous schema.
  // Whole-schema swaps are handled in `replaceSchema` below.
  useEffect(() => {
    setOutput(null);
  }, [yamlText]);

  const formatYaml = () => {
    try {
      setYamlText(formatYamlText(yamlText));
    } catch {
      // Unparseable YAML can't be formatted; the syntax error is already shown.
    }
  };

  // Swapping in a whole schema remounts the preview, so the form starts empty
  // and any payload generated from the previous values is dropped. Both are
  // done here rather than in the effect above, which can't see a load that
  // happens to bring back byte-identical YAML.
  const replaceSchema = (yaml: string) => {
    setYamlText(yaml);
    setPreviewKey((key) => key + 1);
    setOutput(null);
  };

  const handleLoadSchema = (name: string) => {
    const loaded = loadSchema(name);
    if (loaded !== undefined) {
      replaceSchema(loaded);
    }
  };

  const handleResetSchema = () => replaceSchema(schemaYaml);

  return (
    <Box
      ref={containerRef}
      sx={{
        height: 'calc(100vh - 150px)',
        display: 'flex',
        flexDirection: 'row',
        overflow: 'hidden',
        // Room for the toolbar's floating label, which would otherwise be
        // clipped by the hidden overflow.
        pt: 2,
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
          paneTransition(isDragging, 'width'),
        ]}
      >
        <Box
          sx={{
            height: HEADER_HEIGHT,
            flexShrink: 0,
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <SchemaToolbar
            names={names}
            onSave={(name) => saveSchema(name, yamlText)}
            onLoad={handleLoadSchema}
            onDelete={deleteSchema}
            onReset={handleResetSchema}
          />
        </Box>
        <Box sx={{ flex: 1, minHeight: 0, mt: 1 }}>
          <YamlEditorPanel
            yamlText={yamlText}
            diagnostics={diagnostics}
            onChange={setYamlText}
            onFormat={formatYaml}
          />
        </Box>
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
          paneTransition(isDragging, 'backgroundColor'),
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
          paneTransition(isDragging, 'width'),
        ]}
      >
        <Tabs
          value={rightTab}
          onChange={(_, value) => setRightTab(value)}
          sx={{
            height: HEADER_HEIGHT,
            minHeight: HEADER_HEIGHT,
            flexShrink: 0,
            mb: 1,
          }}
        >
          <Tab
            label={Messages.tabs.formPreview}
            sx={{ minHeight: HEADER_HEIGHT }}
          />
          <Tab label={Messages.tabs.output} sx={{ minHeight: HEADER_HEIGHT }} />
        </Tabs>
        {/* Both panes stay mounted and are toggled with `display`: unmounting the
            preview would throw away everything typed into the form, and submit
            switches to the Output tab. */}
        <RoundedBox
          boxProps={{
            sx: {
              width: '100%',
              flex: 1,
              minHeight: 0,
              overflowY: 'auto',
              display: rightTab === 0 ? 'block' : 'none',
            },
          }}
        >
          <TextField
            select
            size="small"
            label={Messages.previewNamespace}
            value={selectedNamespace}
            onChange={(e) => setSelectedNamespace(e.target.value)}
            disabled={namespacesLoading || namespaces.length === 0}
            helperText={Messages.previewNamespaceHelper}
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
                  key={previewKey}
                  schema={parsedSchema}
                  namespace={selectedNamespace}
                  onGenerateOutput={(payload) => {
                    setOutput(payload);
                    setRightTab(1);
                  }}
                />
              </ErrorBoundary>
            </ErrorContextProvider>
          )}
        </RoundedBox>
        <Box
          sx={{
            flex: 1,
            minHeight: 0,
            overflowY: 'auto',
            // Block, not flex: FormCard takes no width props, so a flex parent
            // would shrink the card to its content instead of filling the pane.
            display: rightTab === 1 ? 'block' : 'none',
          }}
        >
          <OutputPanel payload={output} />
        </Box>
      </Box>
    </Box>
  );
};
