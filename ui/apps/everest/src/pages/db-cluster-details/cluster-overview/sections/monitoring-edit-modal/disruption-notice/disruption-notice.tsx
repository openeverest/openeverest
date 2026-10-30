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

import { Alert, Typography } from '@mui/material';
import { Instance } from 'shared-types/api.types';
import { useMonitoringDisruption } from '../hooks/use-monitoring-disruption';
import { Messages } from '../monitoring-edit-modal.messages';

interface DisruptionNoticeProps {
  instance: Instance;
  namespace: string;
}

export const DisruptionNotice = ({
  instance,
  namespace,
}: DisruptionNoticeProps) => {
  const { restarts, singleReplica } = useMonitoringDisruption(
    instance,
    namespace
  );

  if (restarts.length === 0) {
    return null;
  }

  return (
    <Alert
      severity="warning"
      sx={{ mt: 3 }}
      data-testid="monitoring-disruption"
    >
      {restarts.map(({ change, classTitle }) => (
        <Typography key={`${change}-${classTitle}`} variant="body2">
          {Messages.restart(change, classTitle)}
        </Typography>
      ))}
      {singleReplica && (
        <Typography variant="body2">{Messages.singleReplica}</Typography>
      )}
    </Alert>
  );
};
