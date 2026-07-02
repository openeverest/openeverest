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

export const Messages = {
  sharding: {
    invalid: 'Please fill in valid values for sharding',
    min: (val: number) => `The value cannot be less than ${val}`,
    max: (val: number) => `The value cannot be more than ${val}`,
    odd: 'The value cannot be even',
    numberOfShards: 'Nº of shards',
    numberOfConfigurationServers: 'Nº of configuration servers',
    numberOfConfigServersError:
      'The number of configuration servers cannot be 1 if the number of database nodes is greater than 1',
  },
  customValue: 'Custom',
  resourcesCapacityExceeding: (
    fieldName: string,
    value: number | undefined,
    units: string
  ) =>
    `Your specified ${fieldName} size exceeds the ${
      value ? `${value.toFixed(2)} ${units}` : ''
    } available. Enter a smaller value before continuing.`,
  disabledDiskInputTooltip:
    'You can’t change the disk size as the selected storage class doesn’t support volume expansion.',
  descaling: 'Descaling is not allowed',
  upscalingDiskWarning:
    'Disk upscaling is irreversible and may temporarily block further resize actions until complete.',
  integerNumber: 'Disk size must be an integer number.',
  requestsTooltip:
    'Syncing requests and limits grants your database Guaranteed QoS. This prevents unexpected evictions and performance throttling. Disable this only if you require a Burstable configuration and understand the overcommit risks.',
  syncWithLimits: 'Sync requests with limits',
  requestCeiling: (limit: number, units: string) =>
    limit ? `Max: ${limit} ${units} (limit)` : '',
  requestRequired: 'This field is required',
};
