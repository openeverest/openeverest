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

import { Typography } from '@mui/material';
import StatusField from 'components/status-field';
import { READY_TO_STATUS } from '../monitoring-card.constants';
import { Messages } from '../monitoring-card.messages';
import { ConditionStatus } from '../monitoring-card.types';

interface ReadyStatusProps {
  status: ConditionStatus;
}

export const ReadyStatus = ({ status }: ReadyStatusProps) => (
  <StatusField status={status} statusMap={READY_TO_STATUS}>
    <Typography variant="body2">{Messages.ready[status]}</Typography>
  </StatusField>
);
