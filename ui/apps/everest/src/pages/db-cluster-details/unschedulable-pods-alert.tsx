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

import { Alert, AlertTitle, Typography } from '@mui/material';
import type { Instance } from 'shared-types/api.types';
import { Messages } from './db-cluster-details.messages';

export const UnschedulablePodsAlert = ({
  instance,
}: {
  instance?: Instance;
}) => {
  const condition = instance?.status?.conditions?.find(
    (c) => c.type === 'PodsScheduled' && c.status === 'False'
  );

  if (!condition) {
    return null;
  }

  return (
    <Alert
      severity="warning"
      sx={{ my: 1 }}
      data-testid="unschedulable-pods-alert"
    >
      <AlertTitle>{Messages.unschedulablePods.title}</AlertTitle>
      <Typography variant="body2">{Messages.unschedulablePods.hint}</Typography>
      <Typography variant="body2" sx={{ mt: 1, wordBreak: 'break-word' }}>
        {condition.message}
      </Typography>
    </Alert>
  );
};
