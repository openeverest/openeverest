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

import { Box, Stack, Tooltip, Typography } from '@mui/material';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { LabeledContent } from '@percona/ui-lib';
import { usePitrBlock } from './use-pitr-block';
import { WizardPitrStorageRow } from './pitr-storage-row';
import { Messages } from './pitr-block.messages';

export const PitrBlock = () => {
  const {
    supported,
    namespace,
    backupClass,
    storages,
    hasScheduledStorages,
    maxEnabled,
    setEnabled,
    setParameters,
  } = usePitrBlock();

  // PITR needs a backup schedule to stream from; hide the section until one exists.
  if (!supported || !hasScheduledStorages) {
    return null;
  }

  return (
    // V1 spacing: the PITR block sits 56px below the schedules list.
    <Stack sx={{ mt: 7 }}>
      <LabeledContent
        label={Messages.title}
        sx={{ mt: 0 }}
        horizontalStackChildrenSlot={
          <Tooltip title={Messages.description} placement="right" arrow>
            <InfoOutlinedIcon
              fontSize="small"
              sx={{ ml: 0.5, color: 'action.active', cursor: 'help' }}
            />
          </Tooltip>
        }
      >
        <Typography variant="body2">
          {Messages.listLeadIn(maxEnabled)}
        </Typography>
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            mt: 1.5,
            maxWidth: '100%',
          }}
        >
          {storages.map((storage) => (
            <WizardPitrStorageRow
              key={storage.name}
              storage={storage}
              backupClass={backupClass}
              namespace={namespace}
              onToggle={(checked) => setEnabled(storage.name, checked)}
              onSetParameters={(parameters) =>
                setParameters(storage.name, parameters)
              }
            />
          ))}
        </Box>
      </LabeledContent>
    </Stack>
  );
};
