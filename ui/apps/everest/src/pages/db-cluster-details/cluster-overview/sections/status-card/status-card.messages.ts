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
  title: 'Status',
  since: (elapsed: string) => `since ${elapsed}`,
  stale:
    'The details below describe the previous configuration until the provider reports on the current one.',
  none: 'None',
  notReported: 'Not reported yet',
  sections: {
    components: 'Components',
    problems: 'Problems',
    checks: 'Checks',
    inProgress: 'In progress',
    finished: 'Recently finished',
    notices: 'Notices',
  },
  componentReady: (ready: number, total: number) => `${ready}/${total} ready`,
  checksPassing: (count: number) =>
    `${count} check${count === 1 ? '' : 's'} passing`,
  show: 'Show',
  hide: 'Hide',
  nothingInProgress: 'Nothing',
  started: (elapsed: string) => `started ${elapsed} ago`,
  finished: (elapsed: string) => `finished ${elapsed} ago`,
  observed: (elapsed: string) => `for ${elapsed}`,
  allConditions: (count: number) => `All conditions (${count})`,
  noConditions: 'No conditions reported yet.',
  conditions: {
    type: 'Type',
    status: 'Status',
    reason: 'Reason',
    message: 'Message',
    lastTransition: 'Last transition',
    observedGeneration: 'Observed generation',
  },
};
