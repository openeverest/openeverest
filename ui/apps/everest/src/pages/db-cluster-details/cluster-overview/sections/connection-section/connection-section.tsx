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

import { useContext, useState } from 'react';
import { IconButton, TextField } from '@mui/material';
import { CopyToClipboardButton } from '@percona/ui-lib';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import VisibilityOffOutlinedIcon from '@mui/icons-material/VisibilityOffOutlined';
import { HiddenPasswordToggle } from 'components/hidden-row';
import { DbInstanceContext } from 'pages/db-cluster-details/dbCluster.context';
import OverviewSection from '../../overview-section';
import OverviewSectionRow from '../../overview-section-row';
import { Messages } from '../../cluster-overview.messages';
import { ConnectionSectionMessages } from './messages.ts';
import type { ConnectionSectionProps } from './connection-section.types';
import { getConnectionHosts } from './utils.ts';
import ConnectionHost from './connection-host';

const ConnectionSection = ({
  credentials,
  loading,
}: ConnectionSectionProps) => {
  const { canReadCredentials } = useContext(DbInstanceContext);
  const [showUrl, setShowUrl] = useState(false);

  return (
    <OverviewSection
      dataTestId="connection-details"
      title={Messages.titles.connectionDetails}
      loading={loading}
    >
      {credentials ? (
        <>
          <OverviewSectionRow
            label={Messages.fields.host}
            content={getConnectionHosts(credentials.host).map((host) => (
              <ConnectionHost key={host} host={host} />
            ))}
          />
          <OverviewSectionRow
            label={Messages.fields.port}
            content={credentials.port}
          />
          {canReadCredentials && (
            <>
              <OverviewSectionRow
                label={Messages.fields.username}
                content={credentials.username}
              />
              <OverviewSectionRow
                label={Messages.fields.password}
                content={
                  <HiddenPasswordToggle
                    showCopy
                    value={credentials.password || ''}
                  />
                }
              />
            </>
          )}

          {credentials.uri && (
            <TextField
              label={Messages.fields.connectionUrl}
              value={credentials.uri}
              size="small"
              sx={{ maxHeight: '50px', marginTop: '20px', width: '100%' }}
              type={showUrl ? 'text' : 'password'}
              InputProps={{
                endAdornment: (
                  <>
                    <IconButton onClick={() => setShowUrl((s) => !s)}>
                      {showUrl ? (
                        <VisibilityOutlinedIcon />
                      ) : (
                        <VisibilityOffOutlinedIcon />
                      )}
                    </IconButton>
                    <CopyToClipboardButton
                      buttonProps={{
                        sx: { mt: -0.5 },
                        size: 'small',
                      }}
                      textToCopy={credentials.uri || ''}
                    />
                  </>
                ),
              }}
              InputLabelProps={{ shrink: true }}
            />
          )}
        </>
      ) : (
        <OverviewSectionRow
          label={Messages.fields.status}
          content={ConnectionSectionMessages.waitingForInstance}
        />
      )}
    </OverviewSection>
  );
};

export default ConnectionSection;
