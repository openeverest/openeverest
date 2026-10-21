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
import { useLocation, useNavigate } from 'react-router-dom';
import ActionableAlert from 'components/actionable-alert';
import { useGetBackup } from 'hooks/api/backups/useBackups';
import { useClusterName } from 'hooks/api/useClusterName';
import { Messages } from './import-source-banner.messages';
import { ImportSourceBannerProps } from './import-source-banner.types';

export const ImportSourceBanner = ({
  importSource,
}: ImportSourceBannerProps) => {
  const { state } = useLocation();
  const navigate = useNavigate();
  const clusterName = useClusterName();
  const { namespace, dataSource } = importSource;
  const { data: backup } = useGetBackup(
    clusterName,
    namespace,
    dataSource.backupName
  );
  // The user picked the backup by its path.
  const backupLabel =
    backup?.spec.origin.external?.path ?? dataSource.backupName;

  return (
    <Box sx={{ mt: 3 }}>
      <ActionableAlert
        severity="info"
        message={Messages.source(backupLabel, dataSource.storageName)}
        buttonMessage={Messages.changeBackup}
        // The import page restores its choice from the same router state.
        onClick={() => navigate('/databases/import', { state })}
        data-testid="import-source-banner"
      />
    </Box>
  );
};
