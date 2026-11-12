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
import { useNow } from 'hooks/utils/useNow';
import {
  InstanceConditionType,
  InstanceStatusDisplay,
  InstanceStatusSource,
} from 'shared-types/instance.types';
import { formatElapsed, getInstanceCondition } from 'utils/instance-status';
import { Messages, REASON_HINTS } from './status-alert.messages';

interface StatusAlertProps {
  instance: InstanceStatusSource;
  status: InstanceStatusDisplay;
}

// Says what is wrong with the database and whether waiting can still help.
// The problem is whatever NeedsAttention carries, so a new kind of problem
// needs nothing added here to be shown.
export const StatusAlert = ({ instance, status }: StatusAlertProps) => {
  const now = useNow();

  if (status.tone !== 'attention' && status.tone !== 'error') {
    return null;
  }

  const stalled = status.tone === 'error';
  const severity = stalled ? 'error' : 'warning';
  const problem = getInstanceCondition(
    instance,
    InstanceConditionType.NeedsAttention
  );
  const message = problem?.message || status.message;
  const reason = problem?.reason ?? status.reason;
  const hint = Object.prototype.hasOwnProperty.call(REASON_HINTS, reason)
    ? REASON_HINTS[reason]
    : undefined;

  return (
    <Alert
      severity={severity}
      sx={{ mt: 1, mb: 2, '& > .MuiAlert-message': { width: '100%' } }}
      data-testid="instance-status-alert"
    >
      <AlertTitle>{status.label}</AlertTitle>
      {hint && <Box>{hint}</Box>}
      <Box>
        {stalled ? Messages.stalled : Messages.retrying}
        {problem?.lastTransitionTime &&
          ` ${Messages.observed(
            formatElapsed(problem.lastTransitionTime, now)
          )}`}
      </Box>
      {message && (
        <>
          <Box sx={{ mt: 1, fontWeight: 600 }}>{Messages.details}</Box>
          <CodeCopyBlock message={message} severity={severity} />
        </>
      )}
    </Alert>
  );
};
