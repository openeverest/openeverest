import Add from '@mui/icons-material/Add';
import { Button, Typography } from '@mui/material';
import { Table } from '@percona/ui-lib';
import { useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import PoliciesDialog from '../policies-dialog';
import { MRT_ColumnDef } from 'material-react-table';

import DeletePolicyDialog from './delete-policy-dialog';
import { DbEngineType } from '@percona/types';
import {
  useCreatePodSchedulingPolicy,
  useDeletePodSchedulingPolicy,
  usePodSchedulingPolicies,
} from 'hooks';
import { PodSchedulingPolicy } from 'shared-types/affinity.types';
import { humanizeDbType } from 'utils/db';
import { dbEngineToDbType } from '@percona/utils';
import { useQueryClient } from '@tanstack/react-query';
import EmptyState from 'components/empty-state';
import { useRBACPermissions } from 'hooks/rbac';
import PolicyRowActions from './policy-row-actions';
import {
  EVEREST_POLICY_IN_USE_FINALIZER,
  EVEREST_READ_ONLY_FINALIZER,
} from 'consts';

const PoliciesList = () => {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const selectedPolicy = useRef<PodSchedulingPolicy>();
  const { canCreate } = useRBACPermissions('pod-scheduling-policies');
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { mutate: createPolicy } = useCreatePodSchedulingPolicy();
  const { mutate: deletePolicy } = useDeletePodSchedulingPolicy();
  const { data: podSchedulingPolicies = [] } = usePodSchedulingPolicies(
    undefined,
    false,
    {
      refetchInterval: 10000,
    }
  );

  const columns = useMemo<MRT_ColumnDef<PodSchedulingPolicy>[]>(
    () => [
      {
        accessorKey: 'metadata.name',
        header: 'Name',
      },
      {
        accessorKey: 'spec.engineType',
        header: 'Technology',
        Cell: ({ cell }) => {
          const engineType = cell.getValue<DbEngineType>();
          return humanizeDbType(dbEngineToDbType(engineType));
        },
      },
    ],
    []
  );

  const handleOnCreatePolicy = (data: { name: string; type: DbEngineType }) => {
    createPolicy(
      { policyName: data.name, dbType: data.type },
      {
        onSuccess: () => {
          setDialogOpen(false);
          navigate(data.name);
        },
      }
    );
  };

  const handleOnDeleteIconClick = (policy: PodSchedulingPolicy) => {
    selectedPolicy.current = policy;
    setDeleteDialogOpen(true);
  };

  const handleOnDeletePolicy = () => {
    if (selectedPolicy.current) {
      deletePolicy(selectedPolicy.current.metadata.name, {
        onSuccess: () => {
          queryClient.invalidateQueries({
            queryKey: ['pod-scheduling-policies'],
          });
          setDeleteDialogOpen(false);
          selectedPolicy.current = undefined;
        },
      });
    }
  };

  return (
    <>
      <Typography variant="body2" py={2}>
        Manage database pod scheduling across your Kubernetes cluster with
        custom policies for optimal placement and resource efficiency.
      </Typography>
      <Table
        tableName="pod-scheduling-policies"
        emptyState={
          <EmptyState
            onButtonClick={() => setDialogOpen(true)}
            buttonText="Create policy"
            showCreationButton={canCreate}
            contentSlot={
              <Typography variant="body1">
                You currently do not have any policy
              </Typography>
            }
          />
        }
        data={podSchedulingPolicies}
        columns={columns}
        enableRowActions
        enableRowHoverAction
        rowHoverAction={(row) =>
          navigate(
            `/settings/policies/details/pod-scheduling/${row.original.metadata.name}`
          )
        }
        renderTopToolbarCustomActions={() =>
          canCreate ? (
            <Button
              size="small"
              startIcon={<Add />}
              data-testid="add-policy"
              variant="outlined"
              onClick={() => setDialogOpen(true)}
              sx={{ display: 'flex' }}
            >
              Create policy
            </Button>
          ) : null
        }
        renderRowActions={({ row }) => (
          <PolicyRowActions
            policyName={row.original.metadata.name}
            readOnly={row.original.metadata.finalizers.includes(
              EVEREST_READ_ONLY_FINALIZER
            )}
            handleOnDeleteIconClick={() =>
              handleOnDeleteIconClick(row.original)
            }
          />
        )}
      />
      {dialogOpen && (
        <PoliciesDialog
          open
          existingPolicies={podSchedulingPolicies}
          onClose={() => setDialogOpen(false)}
          onSubmit={handleOnCreatePolicy}
        />
      )}
      {deleteDialogOpen && (
        <DeletePolicyDialog
          isOpen
          policyInUse={
            !!selectedPolicy.current?.metadata.finalizers.includes(
              EVEREST_POLICY_IN_USE_FINALIZER
            )
          }
          policyName={selectedPolicy.current?.metadata.name || ''}
          handleCloseDeleteDialog={() => setDeleteDialogOpen(false)}
          handleConfirmDelete={handleOnDeletePolicy}
        />
      )}
    </>
  );
};

export default PoliciesList;
