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

import { beautifyDbTypeName } from '@percona/utils';
import { Messages } from '../../../cluster-overview.messages';
import OverviewSection from '../../../overview-section';
import { BasicInformationOverviewCardProps } from '../../card.types';
import OverviewSectionRow from '../../../overview-section-row';
import { useContext, useMemo, useState } from 'react';
import { UpgradeDbVersionModal } from './upgrade-db-version-modal/upgrade-db-version-modal';
import { useDbVersionsList } from 'components/cluster-form/db-version/useDbVersions';
import { useUpdateDbClusterWithConflictRetry } from 'hooks/api/db-cluster/useUpdateDbCluster';
import { DbClusterContext } from '../../../../dbCluster.context';
import { DbCluster, DbClusterStatus } from 'shared-types/dbCluster.types';
import {
  changeDbClusterVersion,
  mergeNewDbClusterData,
  shouldDbActionsBeBlocked,
} from 'utils/db';
import { useQueryClient } from '@tanstack/react-query';
import { DB_CLUSTER_QUERY } from 'hooks/api';

export const BasicInformationSection = ({
  loading,
  type,
  name,
  namespace,
  version,
}: BasicInformationOverviewCardProps) => {
  const [openEditModal, setOpenEditModal] = useState(false);
  const handleCloseModal = () => {
    setOpenEditModal(false);
  };
  const { dbCluster, canUpdateDb } = useContext(DbClusterContext);
  const queryClient = useQueryClient();
  const { mutate: updateCluster } = useUpdateDbClusterWithConflictRetry(
    dbCluster!,
    {
      onSuccess: (data) => {
        handleCloseModal();
        queryClient.setQueryData<DbCluster>(
          [DB_CLUSTER_QUERY, dbCluster?.metadata.name],
          (oldData) =>
            mergeNewDbClusterData(
              oldData,
              {
                ...data,
                status: { ...data.status, status: DbClusterStatus.upgrading },
              } as DbCluster,
              false
            )
        );
      },
    }
  );
  const upgrading = dbCluster?.status?.status === DbClusterStatus.upgrading;

  const handleSubmit = async (dbVersion: string) => {
    updateCluster(changeDbClusterVersion(dbCluster!, dbVersion));
  };

  const dbVersionsUpgradeList = useDbVersionsList({
    namespace,
    dbType: type,
    currentVersion: version,
  });

  const shouldShowUpgrade = useMemo(() => {
    if (
      dbVersionsUpgradeList &&
      dbCluster?.status?.status === DbClusterStatus.ready
    ) {
      const engineVersions = dbVersionsUpgradeList?.availableVersions?.engine;
      if (engineVersions?.length === 0) {
        return false;
      }
      return !(
        engineVersions?.length === 1 &&
        engineVersions?.find((item) => item.version === version)
      );
    }
    return false;
  }, [dbVersionsUpgradeList, dbCluster?.status?.status, version]);

  const recommendedCRVersion = dbCluster?.status?.recommendedCRVersion!;

  return (
    <OverviewSection
      dataTestId="basic-information"
      title={Messages.titles.basicInformation}
      editText={Messages.actions.upgrade}
      loading={loading}
      {...(shouldShowUpgrade && canUpdateDb && !recommendedCRVersion
        ? {
            actionButtonProps: {
              onClick: () => {
                setOpenEditModal(true);
              },
              disabled: shouldDbActionsBeBlocked(dbCluster?.status?.status),
              children: 'Upgrade',
              'data-testid': 'upgrade-db-btn',
            },
          }
        : undefined)}
    >
      <OverviewSectionRow label="Type" content={beautifyDbTypeName(type)} />
      <OverviewSectionRow label="Name" content={name} />
      <OverviewSectionRow label="Namespace" content={namespace} />
      <OverviewSectionRow label="Version" content={version} />
      {openEditModal && dbVersionsUpgradeList && (
        <UpgradeDbVersionModal
          open={openEditModal}
          handleCloseModal={handleCloseModal}
          handleSubmitModal={handleSubmit}
          dbVersionsUpgradeList={dbVersionsUpgradeList}
          version={version}
          submitting={upgrading}
        />
      )}
    </OverviewSection>
  );
};
