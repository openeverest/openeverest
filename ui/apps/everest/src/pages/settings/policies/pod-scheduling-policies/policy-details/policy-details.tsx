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

import { Alert, Box, Button, Skeleton, Typography } from '@mui/material';
import ArrowBack from '@mui/icons-material/ArrowBack';
import { useNavigate, useParams } from 'react-router-dom';
import {
  usePodSchedulingPolicy,
  useUpdateEntityWithConflictRetry,
} from 'hooks';
import { NoMatch } from 'pages/404/NoMatch';
import { useRef, useState } from 'react';
import { AffinityFormDialog } from '../affinity/affinity-form-dialog/affinity-form-dialog';
import {
  dbPayloadToAffinityRules,
  insertAffinityRuleToExistingPolicy,
  removeRuleInExistingPolicy,
} from 'utils/db';
import { DbType } from '@percona/types';
import { AffinityRule } from 'shared-types/affinity.types';
import PodSchedulingPoliciesTable from 'components/pod-scheduling-policies-table';
import { useRBACPermissionRoute, useRBACPermissions } from 'hooks/rbac';
import {
  EVEREST_POLICY_IN_USE_FINALIZER,
  EVEREST_READ_ONLY_FINALIZER,
} from 'consts';
import { ConfirmDialog } from 'components/confirm-dialog/confirm-dialog';
import { Messages } from '../pod-scheduling-policies.messages';
import { updatePodSchedulingPolicy } from 'api/podSchedulingPolicies';

const PolicyDetails = () => {
  const navigate = useNavigate();
  const { name: policyName = '' } = useParams();
  const [openAffinityDialog, setOpenAffinityDialog] = useState(false);
  const [openRemoveRuleDialog, setOpenRemoveRuleDialog] = useState(false);
  const selectedRule = useRef<AffinityRule>();
  const { canUpdate } = useRBACPermissions(
    'pod-scheduling-policies',
    policyName
  );

  useRBACPermissionRoute([
    {
      action: 'read',
      resource: 'pod-scheduling-policies',
      specificResources: [policyName],
    },
  ]);

  const {
    isLoading,
    isError,
    data: policy,
    refetch: refetchPolicy,
  } = usePodSchedulingPolicy(policyName);

  const { mutate: updatePolicy, isPending: updatingPolicy } =
    useUpdateEntityWithConflictRetry(
      ['pod-scheduling-policy', policyName],
      (newPolicy) => updatePodSchedulingPolicy(newPolicy),
      policy?.metadata.generation || 0,
      refetchPolicy,
      (_, newData) => newData,
      {
        onSuccess: () => {
          setOpenRemoveRuleDialog(false);
          setOpenAffinityDialog(false);
          selectedRule.current = undefined;
        },
      }
    );

  if (isLoading) {
    return (
      <>
        <Skeleton variant="rectangular" />
        <Skeleton variant="rectangular" />
        <Skeleton />
        <Skeleton />
        <Skeleton />
        <Skeleton variant="rectangular" />
      </>
    );
  }

  if (isError || !policy) {
    return <NoMatch />;
  }

  const rules = dbPayloadToAffinityRules(policy);
  const policyInUse = policy.metadata.finalizers.includes(
    EVEREST_POLICY_IN_USE_FINALIZER
  );
  const readOnlyPolicy = policy.metadata.finalizers?.includes(
    EVEREST_READ_ONLY_FINALIZER
  );

  const handleFormSubmit = (rule: AffinityRule) => {
    if (selectedRule.current) {
      // We don't really edit the rule in the object, we just remove the old rule and add the new one
      removeRuleInExistingPolicy(policy, selectedRule.current);
    }
    insertAffinityRuleToExistingPolicy(policy, rule);
    updatePolicy(policy);
  };

  const handleConfirmDelete = () => {
    if (selectedRule.current) {
      removeRuleInExistingPolicy(policy, selectedRule.current);
      updatePolicy(policy);
    }
  };

  const handleOnAddClick = () => {
    selectedRule.current = undefined;
    setOpenAffinityDialog(true);
  };

  return (
    <>
      <Button
        sx={{ mt: 2 }}
        startIcon={<ArrowBack />}
        onClick={() => navigate(-1)}
      >
        Back
      </Button>
      <Box display="flex" alignItems="center" gap={1} mt={3} mb={2}>
        <Typography variant="h6">
          {policyName}
          {policy ? ` / ${String(policy?.spec.engineType ?? '')}` : ''}
        </Typography>
      </Box>
      <PodSchedulingPoliciesTable
        rules={rules}
        canDoChanges={canUpdate && !readOnlyPolicy}
        engineType={policy.spec.engineType}
        onEditClick={(rule) => {
          selectedRule.current = rule;
          setOpenAffinityDialog(true);
        }}
        onDeleteClick={(rule) => {
          selectedRule.current = rule;
          setOpenRemoveRuleDialog(true);
        }}
        onAddRuleClick={handleOnAddClick}
      />
      {openAffinityDialog && (
        <AffinityFormDialog
          isOpen
          submitting={updatingPolicy}
          showInUseWarning={policyInUse}
          dbType={policy.spec.engineType as unknown as DbType}
          handleClose={() => setOpenAffinityDialog(false)}
          handleSubmit={handleFormSubmit}
          defaultValues={selectedRule.current}
        />
      )}
      {openRemoveRuleDialog && (
        <ConfirmDialog
          open
          cancelMessage="Cancel"
          selectedId={''}
          closeModal={() => setOpenRemoveRuleDialog(false)}
          handleConfirm={handleConfirmDelete}
          headerMessage="Delete Rule"
          submitMessage="Delete"
        >
          {policyInUse && (
            <Alert severity="warning">{Messages.policyInUse}</Alert>
          )}
          <Typography px={1} pt={2}>
            Are you sure you want to delete this rule?
          </Typography>
        </ConfirmDialog>
      )}
    </>
  );
};

export default PolicyDetails;
