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

import { MonitoringChange } from './monitoring-edit-modal.types';

const changeVerb: Record<MonitoringChange, string> = {
  [MonitoringChange.Enable]: 'Enabling',
  [MonitoringChange.Remove]: 'Removing',
};

export const Messages = {
  title: 'Edit monitoring',
  save: 'Save',
  restart: (change: MonitoringChange, classTitle: string) =>
    `${changeVerb[change]} ${classTitle} restarts the database pods one at a time; connections briefly drop.`,
  singleReplica: 'This database will be briefly unavailable while it restarts.',
  saved: 'Monitoring updated',
};
