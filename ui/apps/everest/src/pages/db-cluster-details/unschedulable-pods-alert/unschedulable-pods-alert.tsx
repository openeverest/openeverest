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

import { Alert, AlertTitle, Box } from '@mui/material';
import type { InstanceCondition } from 'shared-types/instance.types';
import { Messages } from './unschedulable-pods-alert.messages';

interface UnschedulablePodsAlertProps {
  condition: InstanceCondition;
}

export const UnschedulablePodsAlert = ({
  condition,
}: UnschedulablePodsAlertProps) => (
  <Alert
    severity="warning"
    sx={{ my: 1 }}
    data-testid="unschedulable-pods-alert"
  >
    <AlertTitle>{Messages.title}</AlertTitle>
    {Messages.hint}
    <Box sx={{ mt: 1, wordBreak: 'break-word' }}>
      <strong>{Messages.reasonLabel}</strong> <span>{condition.message}</span>
    </Box>
  </Alert>
);
