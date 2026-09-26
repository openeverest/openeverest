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

import { Box, Typography } from '@mui/material';
import { CopyToClipboardButton } from '@percona/ui-lib';
import { FormCard } from 'components/form-card';
import { Messages } from './output-panel.messages';

export type OutputPanelProps = {
  payload: Record<string, unknown> | null;
};

export const OutputPanel = ({ payload }: OutputPanelProps): JSX.Element => {
  const json = payload === null ? '' : JSON.stringify(payload, null, 2);

  return (
    <FormCard
      title={Messages.title}
      controlComponent={
        payload !== null && (
          <CopyToClipboardButton
            textToCopy={json}
            showCopyButtonText
            copyCommand={Messages.copy}
            buttonProps={{ size: 'small' }}
          />
        )
      }
      cardContent={
        payload === null ? (
          <Typography variant="body2" color="text.secondary">
            {Messages.emptyState}
          </Typography>
        ) : (
          <Box
            component="pre"
            data-testid="output-json"
            sx={{
              fontFamily: 'monospace',
              fontSize: '0.8rem',
              whiteSpace: 'pre',
              overflowX: 'auto',
              m: 0,
            }}
          >
            {json}
          </Box>
        )
      }
    />
  );
};
