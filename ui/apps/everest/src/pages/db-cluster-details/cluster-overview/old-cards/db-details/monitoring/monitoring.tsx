// everest
// Copyright (C) 2023 Percona LLC
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

import { Messages } from '../../../cluster-overview.messages';
import OverviewSection from '../../../overview-section';
import { MonitoringConfigurationOverviewCardProps } from '../../card.types';
import OverviewSectionRow from '../../../overview-section-row';
import { useState } from 'react';
import { MonitoringEditModal } from './edit-monitoring';
// import { DbClusterContext } from 'pages/db-cluster-details/dbCluster.context';
// import { useUpdateDbClusterWithConflictRetry } from 'hooks/api/db-cluster/useUpdateDbCluster';
// import { changeDbClusterMonitoring } from 'utils/db';
import { Link } from 'react-router-dom';

export const MonitoringDetails = ({
  loading,
  monitoring,
}: MonitoringConfigurationOverviewCardProps) => {
  const [openEditModal, setOpenEditModal] = useState(false);
  // const { dbCluster, canUpdateDb } = useContext(DbClusterContext);
  // const editable =
  //   canUpdateDb && !shouldDbActionsBeBlocked(dbCluster?.status?.status);

  // const { mutate: updateCluster } = useUpdateDbClusterWithConflictRetry(
  //   dbCluster!,
  //   {
  //     onSuccess: () => handleCloseModal(),
  //   }
  // );

  const handleCloseModal = () => {
    setOpenEditModal(false);
  };

  const handleSubmit = () => {
    // updateCluster(
    //   changeDbClusterMonitoring(
    //     dbCluster!,
    //     enabled ? monitoringName : undefined
    //   )
    // );
  };

  return (
    <OverviewSection
      dataTestId="monitoring"
      title={Messages.titles.monitoring}
      loading={loading}
      actionButtonProps={{
        onClick: () => {
          setOpenEditModal(true);
        },
        'data-testid': 'edit-monitoring-db-btn',
      }}
      // editable={editable}
    >
      <OverviewSectionRow
        label={Messages.fields.status}
        content={
          monitoring ? Messages.fields.enabled : Messages.fields.disabled
        }
      />
      {monitoring && (
        <OverviewSectionRow
          label={Messages.fields.name}
          content={
            <Link to="/settings/monitoring-endpoints">{monitoring}</Link>
          }
        />
      )}

      {openEditModal && (
        <MonitoringEditModal
          open={openEditModal}
          handleCloseModal={handleCloseModal}
          handleSubmitModal={handleSubmit}
          enabled={!!monitoring}
          monitoringName={monitoring}
        />
      )}
    </OverviewSection>
  );
};
