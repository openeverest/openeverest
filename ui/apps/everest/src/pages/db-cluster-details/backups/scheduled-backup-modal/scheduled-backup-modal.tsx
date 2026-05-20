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

// TODO: Migrate scheduled backup modal to v2 Instance API — check main for full implementation
// Original v1 implementation (from main):
// import { useContext } from 'react';
// import { ScheduleFormDialog } from 'components/schedule-form-dialog/schedule-form-dialog';
// import { ScheduleModalContext } from '../backups.context';
// import { useUpdateDbClusterWithConflictRetry } from 'hooks/api/db-cluster/useUpdateDbCluster';
// import { FormMode } from 'components/ui-generator/ui-generator.types';
//
// export const ScheduledBackupModal = () => {
//   const {
//     mode,
//     selectedScheduleName,
//     openScheduleModal,
//     setOpenScheduleModal,
//     dbCluster,
//   } = useContext(ScheduleModalContext);
//
//   const { mutate: updateDbCluster } = useUpdateDbClusterWithConflictRetry(
//     dbCluster?.metadata?.name ?? '',
//     dbCluster?.metadata?.namespace ?? ''
//   );
//
//   const handleSubmit = (data) => { ... };
//   const handleClose = () => setOpenScheduleModal(false);
//
//   if (!openScheduleModal || !dbCluster) return null;
//
//   return (
//     <ScheduleFormDialog
//       mode={mode}
//       dbCluster={dbCluster}
//       selectedScheduleName={selectedScheduleName}
//       onSubmit={handleSubmit}
//       onClose={handleClose}
//     />
//   );
// };
export const ScheduledBackupModal = () => {
  return null;
};
