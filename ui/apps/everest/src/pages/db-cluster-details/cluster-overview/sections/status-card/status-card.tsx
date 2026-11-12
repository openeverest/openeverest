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

import { useMemo, useState } from 'react';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import {
  Box,
  Button,
  Collapse,
  Divider,
  Stack,
  Typography,
} from '@mui/material';
import { OverviewCard } from '@percona/ui-lib';
import {
  InstanceStatus,
  InstanceStatusSince,
  useInstanceStatus,
} from 'components/instance-status';
import { Instance } from 'shared-types/api.types';
import {
  getInstanceStatusDetails,
  InstanceConditionEntry,
} from 'utils/instance-conditions';
import {
  getInstanceStatusExplanation,
  getReasonLabel,
} from 'utils/instance-status';
import { ConditionsTable } from './conditions-table';
import {
  CheckIcon,
  FinishedIcon,
  NoticeIcon,
  StatusEmpty,
  StatusLine,
  StatusRow,
} from './status-card-rows';
import { Messages } from './status-card.messages';

interface StatusCardProps {
  instance: Instance;
}

const testId = (section: string, entry: InstanceConditionEntry) =>
  `status-${section}-${entry.condition.type}`;

// How the database is doing, from the one line every client shows down to
// the conditions it is read from: whether it is up and what is happening,
// what is wrong, what is still being checked, what is running, and what
// only wants a decision.
export const StatusCard = ({ instance }: StatusCardProps) => {
  const status = useInstanceStatus(instance);
  const details = useMemo(() => getInstanceStatusDetails(instance), [instance]);
  const explanation = getInstanceStatusExplanation(status);
  const [showPassing, setShowPassing] = useState(false);
  const [showConditions, setShowConditions] = useState(false);

  const components = instance.status?.components ?? [];
  const {
    problems,
    passingChecks,
    openChecks,
    runningOperations,
    finishedOperations,
    notices,
    conditions,
  } = details;
  const hasChecks = passingChecks.length + openChecks.length > 0;

  return (
    <OverviewCard
      dataTestId="status-details"
      cardHeaderProps={{ title: Messages.title }}
      sx={{ mb: 2 }}
    >
      <Stack sx={{ gap: 2 }}>
        <Box data-testid="status-summary">
          <Stack
            direction="row"
            sx={{
              flexWrap: 'wrap',
              justifyContent: 'space-between',
              alignItems: 'center',
              columnGap: 2,
            }}
          >
            <Typography variant="subtitle1" component="div">
              <InstanceStatus
                status={status}
                showAvailability
                dataTestId="status-card"
              />
            </Typography>
            {status.since && (
              <InstanceStatusSince since={status.since}>
                {Messages.since}
              </InstanceStatusSince>
            )}
          </Stack>
          {explanation && (
            <Typography
              variant="body2"
              color="text.secondary"
              sx={{ mt: 0.5, wordBreak: 'break-word', whiteSpace: 'pre-line' }}
              data-testid="status-card-message"
            >
              {explanation}
            </Typography>
          )}
        </Box>
        <Divider />
        {status.stale && conditions.length > 0 && (
          <Typography
            variant="body2"
            color="text.secondary"
            sx={{ fontStyle: 'italic' }}
            data-testid="status-stale"
          >
            {Messages.stale}
          </Typography>
        )}

        <StatusRow label={Messages.sections.components} dataTestId="components">
          {components.length === 0 ? (
            <StatusEmpty>{Messages.notReported}</StatusEmpty>
          ) : (
            <Stack direction="row" sx={{ flexWrap: 'wrap', columnGap: 4 }}>
              {components.map(({ name, replicas, readyReplicas }) => (
                <Typography
                  key={name}
                  variant="body2"
                  data-testid={`status-component-${name}`}
                >
                  {name}{' '}
                  <Box component="span" sx={{ color: 'text.secondary' }}>
                    {Messages.componentReady(readyReplicas ?? 0, replicas ?? 0)}
                  </Box>
                </Typography>
              ))}
            </Stack>
          )}
        </StatusRow>

        <StatusRow label={Messages.sections.problems} dataTestId="problems">
          {problems.length === 0 ? (
            <StatusEmpty>{Messages.none}</StatusEmpty>
          ) : (
            problems.map((entry) => {
              const reason = getReasonLabel(entry.condition.reason);
              return (
                <StatusLine
                  key={entry.condition.type}
                  icon={<CheckIcon status="False" />}
                  title={entry.title}
                  // A problem no check stands for is titled by its reason.
                  reason={reason === entry.title ? undefined : reason}
                  message={entry.condition.message}
                  dataTestId={testId('problem', entry)}
                />
              );
            })
          )}
        </StatusRow>

        <StatusRow label={Messages.sections.checks} dataTestId="checks">
          {!hasChecks && <StatusEmpty>{Messages.notReported}</StatusEmpty>}
          {passingChecks.length > 0 && (
            <>
              <StatusLine
                icon={<CheckIcon status="True" />}
                title={Messages.checksPassing(passingChecks.length)}
                action={
                  <Button
                    size="small"
                    sx={{ py: 0, minWidth: 0 }}
                    aria-expanded={showPassing}
                    onClick={() => setShowPassing((shown) => !shown)}
                    data-testid="status-checks-toggle"
                  >
                    {showPassing ? Messages.hide : Messages.show}
                  </Button>
                }
                dataTestId="status-checks-passing"
              />
              <Collapse in={showPassing} unmountOnExit>
                <Stack sx={{ gap: 0.75, pl: 3 }}>
                  {passingChecks.map((entry) => (
                    <StatusLine
                      key={entry.condition.type}
                      icon={<CheckIcon status="True" />}
                      title={entry.title}
                      dataTestId={testId('check', entry)}
                    />
                  ))}
                </Stack>
              </Collapse>
            </>
          )}
          {openChecks.map((entry) => (
            <StatusLine
              key={entry.condition.type}
              icon={<CheckIcon status={entry.condition.status} />}
              title={entry.title}
              // What a failing check says is listed with the problems.
              message={
                entry.condition.status === 'False'
                  ? getReasonLabel(entry.condition.reason)
                  : entry.condition.message ||
                    getReasonLabel(entry.condition.reason)
              }
              dataTestId={testId('check', entry)}
            />
          ))}
        </StatusRow>

        <StatusRow
          label={Messages.sections.inProgress}
          dataTestId="in-progress"
        >
          {runningOperations.length === 0 ? (
            <StatusEmpty>{Messages.nothingInProgress}</StatusEmpty>
          ) : (
            runningOperations.map((entry) => (
              <StatusLine
                key={entry.condition.type}
                icon={<CheckIcon status="Unknown" />}
                title={entry.title}
                message={entry.condition.message}
                trailing={
                  <InstanceStatusSince
                    since={entry.condition.lastTransitionTime}
                  >
                    {Messages.started}
                  </InstanceStatusSince>
                }
                dataTestId={testId('operation', entry)}
              />
            ))
          )}
        </StatusRow>

        {finishedOperations.length > 0 && (
          <StatusRow label={Messages.sections.finished} dataTestId="finished">
            {finishedOperations.map((entry) => (
              <StatusLine
                key={entry.condition.type}
                icon={<FinishedIcon />}
                title={entry.title}
                reason={getReasonLabel(entry.condition.reason)}
                message={entry.condition.message}
                trailing={
                  <InstanceStatusSince
                    since={entry.condition.lastTransitionTime}
                  >
                    {Messages.finished}
                  </InstanceStatusSince>
                }
                dataTestId={testId('operation', entry)}
              />
            ))}
          </StatusRow>
        )}

        <StatusRow label={Messages.sections.notices} dataTestId="notices">
          {notices.length === 0 ? (
            <StatusEmpty>{Messages.none}</StatusEmpty>
          ) : (
            notices.map((entry) => (
              <StatusLine
                key={entry.condition.type}
                icon={<NoticeIcon />}
                title={entry.title}
                message={entry.condition.message}
                dataTestId={testId('notice', entry)}
              />
            ))
          )}
        </StatusRow>

        <Box>
          <Button
            size="small"
            aria-expanded={showConditions}
            onClick={() => setShowConditions((shown) => !shown)}
            startIcon={
              <KeyboardArrowDownIcon
                sx={{
                  transform: showConditions ? 'none' : 'rotate(-90deg)',
                  transition: (theme) => theme.transitions.create('transform'),
                }}
              />
            }
            sx={{ ml: -0.5 }}
            data-testid="status-conditions-toggle"
          >
            {Messages.allConditions(conditions.length)}
          </Button>
          <Collapse in={showConditions} unmountOnExit>
            <ConditionsTable conditions={conditions} />
          </Collapse>
        </Box>
      </Stack>
    </OverviewCard>
  );
};
