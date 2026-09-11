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

import { Button, MenuItem, Stack, TextField, Tooltip } from '@mui/material';
import { TextInput } from '@percona/ui-lib';
import { FormDialog } from 'components/form-dialog';
import { useState } from 'react';
import { z } from 'zod';
import { Messages } from './schema-toolbar.messages';

const saveSchemaFormSchema = z.object({
  name: z.string().trim().min(1),
});

type SaveSchemaForm = z.infer<typeof saveSchemaFormSchema>;

export type SchemaToolbarProps = {
  names: string[];
  // The parent supplies the YAML; the toolbar only knows the name.
  onSave: (name: string) => void;
  onLoad: (name: string) => void;
  onDelete: (name: string) => void;
  onReset: () => void;
};

// Presentational only: every action is a callback, no persistence here.
export const SchemaToolbar = ({
  names,
  onSave,
  onLoad,
  onDelete,
  onReset,
}: SchemaToolbarProps) => {
  const [selectedName, setSelectedName] = useState('');
  const [saveOpen, setSaveOpen] = useState(false);

  const canDelete = selectedName !== '' && names.includes(selectedName);

  const handleLoad = (name: string) => {
    setSelectedName(name);
    onLoad(name);
  };

  const closeSaveDialog = () => setSaveOpen(false);

  const confirmSave = ({ name }: SaveSchemaForm) => {
    onSave(name);
    setSelectedName(name);
    setSaveOpen(false);
  };

  const handleDelete = () => {
    if (!canDelete) {
      return;
    }
    onDelete(selectedName);
    setSelectedName('');
  };

  const handleReset = () => {
    onReset();
    setSelectedName('');
  };

  return (
    // One line only: wrapping would misalign this strip with the preview tabs.
    <Stack
      direction="row"
      spacing={1}
      alignItems="center"
      flexWrap="nowrap"
      sx={{ width: '100%', overflowX: 'auto', py: 0.5 }}
    >
      <TextField
        select
        size="small"
        label={Messages.savedSchemas}
        value={selectedName}
        onChange={(e) => handleLoad(e.target.value)}
        disabled={names.length === 0}
        sx={{ minWidth: 190, flexShrink: 1 }}
      >
        {names.map((name) => (
          <MenuItem key={name} value={name}>
            {name}
          </MenuItem>
        ))}
      </TextField>
      <Button
        size="small"
        variant="contained"
        onClick={() => setSaveOpen(true)}
        sx={{ flexShrink: 0 }}
      >
        {Messages.save}
      </Button>
      <Button
        size="small"
        variant="outlined"
        onClick={handleDelete}
        disabled={!canDelete}
        sx={{ flexShrink: 0 }}
      >
        {Messages.delete}
      </Button>
      <Tooltip title={Messages.resetTooltip}>
        <Button
          size="small"
          variant="text"
          onClick={handleReset}
          sx={{ flexShrink: 0, whiteSpace: 'nowrap' }}
        >
          {Messages.reset}
        </Button>
      </Tooltip>

      {/* Mounted only while open so the name field picks up the current
          selection as its default. */}
      {saveOpen && (
        <FormDialog
          isOpen
          closeModal={closeSaveDialog}
          headerMessage={Messages.saveDialog.header}
          schema={saveSchemaFormSchema}
          defaultValues={{ name: selectedName }}
          onSubmit={confirmSave}
          submitMessage={Messages.save}
          cancelMessage={Messages.cancel}
        >
          <TextInput
            name="name"
            label={Messages.saveDialog.nameLabel}
            textFieldProps={{ autoFocus: true }}
          />
        </FormDialog>
      )}
    </Stack>
  );
};
