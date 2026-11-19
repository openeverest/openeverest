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
import { InstanceStatusSource } from 'shared-types/instance.types';
import { getInstanceProblem } from 'utils/instance-status';
import { Messages, REASON_HINTS } from './status-alert.messages';

interface StatusAlertProps {
  instance?: InstanceStatusSource;
}

// Says what is wrong with the Instance and whether waiting can still help.
export const StatusAlert = ({ instance }: StatusAlertProps) => {
  const problem = getInstanceProblem(instance);

  if (!problem) {
    return null;
  }

  const severity = problem.outlook === 'stalled' ? 'error' : 'warning';
  const outlook =
    problem.outlook === 'unattended'
      ? undefined
      : Messages.outlook[problem.outlook];
  const hint = Object.prototype.hasOwnProperty.call(
    REASON_HINTS,
    problem.reason
  )
    ? REASON_HINTS[problem.reason]
    : undefined;

  return (
    <Alert
      severity={severity}
      sx={{ mt: 1, mb: 2, '& > .MuiAlert-message': { width: '100%' } }}
      data-testid="instance-status-alert"
    >
      <AlertTitle>{problem.label}</AlertTitle>
      {hint && <Box>{hint}</Box>}
      {outlook && <Box>{outlook}</Box>}
      {problem.message && (
        <>
          <Box sx={{ mt: 1, fontWeight: 600 }}>{Messages.details}</Box>
          <CodeCopyBlock message={problem.message} severity={severity} />
        </>
      )}
    </Alert>
  );
};
