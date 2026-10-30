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

import { MAX_MONITORING_DESTINATIONS } from './monitoring-step.constants';
import {
  DestinationDisabledReason,
  DestinationReadiness,
} from './monitoring-step.types';

export const Messages = {
  title: 'Monitoring',
  caption:
    'Choose where this database sends its monitoring data. You can send it to several systems at once.',
  noDestinations: (namespace: string) =>
    `No destinations of this system in the ${namespace} namespace yet.`,
  readiness: {
    [DestinationReadiness.Ready]: 'Ready',
    [DestinationReadiness.Pending]: 'Waiting for its controller',
    [DestinationReadiness.NotReady]: 'Not ready',
  } satisfies Record<DestinationReadiness, string>,
  disabledReason: {
    [DestinationDisabledReason.LimitReached]: `A database can send to at most ${MAX_MONITORING_DESTINATIONS} destinations`,
    [DestinationDisabledReason.IntegrationTaken]:
      'Another destination of this system is already selected',
  } satisfies Record<DestinationDisabledReason, string>,
  entryParameters: (destinationName: string) =>
    `Settings for ${destinationName}`,
};
