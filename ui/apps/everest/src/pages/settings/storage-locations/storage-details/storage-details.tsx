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

import { Box } from '@mui/material';
import { formatDisplayValue } from 'components/ui-generator/utils/object-path';
import { BackupStorageCRD } from 'shared-types/backupStorages.types';
import { ExpandedRowInfoLine } from '../ExpandedRowInfoLine/ExpandedRowInfoLine';
import { Messages } from '../storage-locations.messages';
import { describeBackupStorageTls } from '../storage-locations.utils';

interface StorageDetailsProps {
  storage: BackupStorageCRD;
}

// How the storage connects: its credentials, TLS and the parameters of its
// class, e.g. an S3 endpoint and region.
export const StorageDetails = ({
  storage: {
    spec: { authentication, tls, parameters = {} },
  },
}: StorageDetailsProps) => (
  <Box sx={{ display: 'flex', alignItems: 'start', gap: '50px' }}>
    <Box>
      <ExpandedRowInfoLine
        label={Messages.details.authMode}
        value={authentication.mode}
      />
      <ExpandedRowInfoLine
        label={Messages.details.credentials}
        value={
          authentication.credentialsSecretRef?.name ?? Messages.details.none
        }
      />
      <ExpandedRowInfoLine
        label={Messages.details.tls}
        value={describeBackupStorageTls(tls)}
      />
    </Box>
    <Box>
      {Object.entries(parameters).map(([key, value]) => (
        <ExpandedRowInfoLine
          key={key}
          label={key}
          value={formatDisplayValue(value)}
        />
      ))}
    </Box>
  </Box>
);
