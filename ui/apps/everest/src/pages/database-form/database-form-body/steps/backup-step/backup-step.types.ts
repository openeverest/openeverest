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

// Wizard PITR form state: per-storage PITR config keyed by storage name. This
// is the flat working shape of the form (like backup.schedules), not the nested
// Instance spec — buildBackupSpecFromWizard maps each entry onto the matching
// storages[i].pitr at submit.
export interface WizardPitrConfig {
  enabled: boolean;
  parameters?: Record<string, unknown>;
}

export type WizardPitrMap = Record<string, WizardPitrConfig>;

// Nested Instance spec.backup shape produced from the wizard's flat form state
// by buildBackupSpecFromWizard and sent to the API on submit.
export interface WizardBackupSpec {
  classRef: { name: string };
  enabled: boolean;
  storages: Array<{
    name: string;
    storageRef: { name: string };
    schedules: Array<{
      name: string;
      cron: string;
      enabled: boolean;
      retentionCopies?: number;
      parameters?: Record<string, unknown>;
    }>;
    pitr?: { enabled: boolean; parameters?: Record<string, unknown> };
  }>;
}
