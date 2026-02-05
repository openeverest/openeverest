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
import { AdvancedConfigurationOverviewCardProps } from '../../card.types';
import OverviewSectionRow from '../../../overview-section-row';
import { DbClusterContext } from 'pages/db-cluster-details/dbCluster.context';
import { useContext, useState } from 'react';
import { AdvancedConfigurationEditModal } from './edit-advanced-configuration';
import { useUpdateDbClusterWithConflictRetry } from 'hooks';
import { AdvancedConfigurationFormType } from 'components/cluster-form/advanced-configuration/advanced-configuration-schema';
import {
  changeDbClusterAdvancedConfig,
  shouldDbActionsBeBlocked,
} from 'utils/db';
import { Link } from 'react-router-dom';
import { useRBACPermissions } from 'hooks/rbac';
import { EMPTY_LOAD_BALANCER_CONFIGURATION } from 'consts';
import { Alert, IconButton, Typography } from '@mui/material';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import SplitHorizonDomainsTable from './split-horizon-domains-table';
import { ConfirmDialog } from 'components/confirm-dialog/confirm-dialog';

export const AdvancedConfiguration = ({
  loading,
  externalAccess,
  parameters,
  storageClass,
  podSchedulingPolicy,
  loadBalancerConfig,
  splitHorizonDNS,
  splitHorizonDomains,
}: AdvancedConfigurationOverviewCardProps) => {
  const {
    canUpdateDb,
    dbCluster,
    queryResult: { refetch },
  } = useContext(DbClusterContext);
  const [openEditModal, setOpenEditModal] = useState(false);
  const [showSplitHorizonDomainsTable, setShowSplitHorizonDomainsTable] =
    useState(false);
  const [updating, setUpdating] = useState(false);
  const { canRead: canReadPolicy } = useRBACPermissions(
    'pod-scheduling-policies',
    podSchedulingPolicy
  );

  const { mutate: updateCluster } = useUpdateDbClusterWithConflictRetry(
    dbCluster!,
    {
      onSuccess: async () => {
        await refetch();
        handleCloseModal();
        setUpdating(false);
      },
      onError: () => setUpdating(false),
    }
  );
  const handleCloseModal = () => {
    setOpenEditModal(false);
  };
  const editable =
    canUpdateDb && !shouldDbActionsBeBlocked(dbCluster?.status?.status);

  const handleSubmit = async ({
    sourceRanges,
    engineParametersEnabled,
    engineParameters,
    podSchedulingPolicyEnabled,
    podSchedulingPolicy,
    exposureMethod,
    loadBalancerConfigName,
    splitHorizonDNS,
  }: AdvancedConfigurationFormType) => {
    setUpdating(true);
    updateCluster(
      changeDbClusterAdvancedConfig(
        dbCluster!,
        engineParametersEnabled,
        exposureMethod,
        engineParameters,
        sourceRanges,
        podSchedulingPolicyEnabled,
        podSchedulingPolicy,
        loadBalancerConfigName,
        splitHorizonDNS
      )
    );
  };

  return (
    <OverviewSection
      title={Messages.titles.advancedConfiguration}
      loading={loading}
      dataTestId="advanced-configuration"
      editable={editable}
      {...(canUpdateDb
        ? {
            actionButtonProps: {
              onClick: () => {
                setOpenEditModal(true);
              },
              children: Messages.actions.edit,
              'data-testid': 'edit-advanced-configuration-db-btn',
            },
          }
        : undefined)}
    >
      <OverviewSectionRow
        label={Messages.fields.externalAccess}
        content={
          externalAccess ? Messages.fields.enabled : Messages.fields.disabled
        }
      />
      <OverviewSectionRow
        label={Messages.fields.parameters}
        content={
          parameters ? Messages.fields.enabled : Messages.fields.disabled
        }
      />
      <OverviewSectionRow
        label={Messages.fields.storageClass}
        content={storageClass}
      />

      <OverviewSectionRow
        label={Messages.fields.podSchedulingPolicy}
        content={
          podSchedulingPolicy ? (
            canReadPolicy ? (
              <Link
                to={`/settings/policies/details/pod-scheduling/${podSchedulingPolicy}`}
              >
                {podSchedulingPolicy}
              </Link>
            ) : (
              podSchedulingPolicy
            )
          ) : (
            Messages.fields.disabled
          )
        }
      />
      <OverviewSectionRow
        label={Messages.fields.loadBalancerConfig}
        content={
          loadBalancerConfig ? (
            canReadPolicy &&
            loadBalancerConfig !== EMPTY_LOAD_BALANCER_CONFIGURATION ? (
              <Link
                to={`/settings/policies/details/load-balancer-configuration/${loadBalancerConfig}`}
              >
                {loadBalancerConfig}
              </Link>
            ) : (
              loadBalancerConfig
            )
          ) : (
            Messages.fields.disabled
          )
        }
      />
      <OverviewSectionRow
        label={Messages.fields.splitHorizonDNS}
        content={
          <Typography variant="body2">
            {splitHorizonDNS || Messages.fields.disabled}
            <IconButton onClick={() => setShowSplitHorizonDomainsTable(true)}>
              <VisibilityOutlinedIcon color="primary" fontSize="small" />
            </IconButton>
          </Typography>
        }
      />
      {openEditModal && dbCluster && (
        <AdvancedConfigurationEditModal
          open={openEditModal}
          handleCloseModal={handleCloseModal}
          handleSubmitModal={handleSubmit}
          dbCluster={dbCluster}
          submitting={updating}
        />
      )}
      {showSplitHorizonDomainsTable && (
        <ConfirmDialog
          selectedId="split-horizon-domains"
          closeModal={() => setShowSplitHorizonDomainsTable(false)}
          handleConfirm={() => setShowSplitHorizonDomainsTable(false)}
          submitMessage="Close"
          headerMessage="Domains"
          fullWidth
          maxWidth="lg"
          open
        >
          <Alert severity="info" sx={{ mb: 2 }}>
            <Typography variant="body2">
              It might take several minutes for public IPs to be updated.
            </Typography>
          </Alert>
          <SplitHorizonDomainsTable domains={splitHorizonDomains || []} />
        </ConfirmDialog>
      )}
    </OverviewSection>
  );
};
