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

export interface RestoreDbModalProps {
  isOpen: boolean;
  closeModal: () => void;
  instanceName: string;
  namespace: string;
  isNewClusterMode?: boolean;
  preselectedBackupName?: string;
}

export interface RestorableBackupOption {
  name: string;
  startedAt?: string;
}

export interface ModalContentProps {
  isLoading: boolean;
  header: string;
  succeededBackups: RestorableBackupOption[];
  // TODO: Re-enable PITR props when PITR restore flow is implemented.
  // pitrData?: DatabaseClusterPitr;
  // backupName?: string;
}
