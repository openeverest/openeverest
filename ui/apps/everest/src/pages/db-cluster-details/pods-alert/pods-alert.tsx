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
import { CodeCopyBlock } from '@percona/ui-lib';
import type { PodsAlertCondition } from 'shared-types/instance.types';
import { Messages } from './pods-alert.messages';

interface PodsAlertProps {
  condition: PodsAlertCondition;
}

export const PodsAlert = ({ condition }: PodsAlertProps) => {
  const { title, hint } = Messages.alerts[condition.reason];

  return (
    <Alert
      severity="warning"
      sx={{ mt: 1, mb: 2, '& > .MuiAlert-message': { width: '100%' } }}
      data-testid="pods-alert"
    >
      <AlertTitle>{title}</AlertTitle>
      {hint}
      <Box sx={{ mt: 1, fontWeight: 600 }}>{Messages.reasonLabel}</Box>
      <CodeCopyBlock message={condition.message} severity="warning" />
    </Alert>
  );
};
