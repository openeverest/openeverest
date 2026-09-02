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

import { useMemo, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';
import BackNavigationText from 'components/back-navigation-text';
import { useNamespace } from 'hooks/api/namespaces';
import {
  useDbEngines,
  useOperatorUpgrade,
  useOperatorsUpgradePlan,
} from 'hooks/api/db-engines';
import { useOperatorUpgradePhase } from 'hooks/api/db-engines/useOperatorUpgradePhase';
import { DbEngineStatus } from 'shared-types/dbEngines.types';
import { useRBACPermissions } from 'hooks/rbac';
import { NoMatch } from 'pages/404/NoMatch';
import UpgradeHeader from './upgrade-header';
import ClusterStatusTable from './cluster-status-table';
import UpgradeModal from './upgrade-modal';
import OperatorVersionsHeader from './operator-versions-header';

const NamespaceDetails = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [modalOpen, setModalOpen] = useState(false);
  const { namespace: namespaceName = '' } = useParams();
  const { data: namespace, isLoading: loadingNamespace } = useNamespace(
    namespaceName,
    {
      enabled: !!namespaceName,
    }
  );
  const { data: dbEngines = [] } = useDbEngines(
    namespaceName,
    {
      enabled: !!namespace,
      refetchInterval: 2 * 1000,
    },
    true
  );
  const anyEngineUpgrading = dbEngines.some(
    (e) => e.status === DbEngineStatus.UPGRADING
  );
  // Track previous pending-actions state to set a faster refetch interval
  // without creating a circular dependency on the query result itself.
  const hasPendingActionsRef = useRef(false);
  const { data: operatorsUpgradePlan, isLoading: loadingOperatorsUpgradePlan } =
    useOperatorsUpgradePlan(namespaceName, dbEngines, {
      initialData: {
        upgrades: [],
        pendingActions: [],
        upToDate: [],
      },
      enabled: !!namespace && dbEngines.length > 0,
      refetchInterval:
        anyEngineUpgrading || hasPendingActionsRef.current
          ? 2 * 1000
          : 5 * 1000,
    });
  hasPendingActionsRef.current = (
    operatorsUpgradePlan?.pendingActions || []
  ).some((a) => a.pendingTask !== 'ready');
  const phase = useOperatorUpgradePhase(dbEngines, operatorsUpgradePlan);
  const operatorNamesWithUpgrades = useMemo(
    () =>
      (operatorsUpgradePlan?.upgrades || []).map(
        (upgrade) => `${namespaceName}/${upgrade.name}`
      ) || [],
    [namespaceName, operatorsUpgradePlan]
  );

  const { mutate: upgradeOperator } = useOperatorUpgrade(namespaceName);
  const { canUpdate } = useRBACPermissions(
    'database-engines',
    operatorNamesWithUpgrades
  );

  const performUpgrade = () => {
    upgradeOperator(null, {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: ['dbEngines', namespaceName],
        });
        queryClient.invalidateQueries({
          queryKey: ['operatorUpgradePlan', namespace],
        });
        setModalOpen(false);
      },
    });
  };

  if (loadingNamespace || loadingOperatorsUpgradePlan) {
    return null;
  }

  if (!namespace) {
    return <NoMatch />;
  }

  return (
    <>
      <BackNavigationText
        text={namespaceName}
        onBackClick={() => navigate('/settings/namespaces')}
        rightSlot={
          <OperatorVersionsHeader
            operatorsUpgradePlan={operatorsUpgradePlan!}
          />
        }
      />
      <UpgradeHeader
        onUpgrade={() => setModalOpen(true)}
        upgradeAvailable={!!operatorsUpgradePlan?.upgrades.length}
        upgradeAllowed={canUpdate}
        pendingUpgradeTasks={
          !!operatorsUpgradePlan?.pendingActions.some(
            (action) => action.pendingTask !== 'ready'
          )
        }
        upgradePhase={phase}
        mt={3}
      />
      <ClusterStatusTable
        namespace={namespaceName}
        pendingActions={operatorsUpgradePlan?.pendingActions || []}
        dbEngines={dbEngines}
        upgradePhase={phase}
      />
      <UpgradeModal
        namespace={namespace}
        operatorsUpgradeTasks={operatorsUpgradePlan?.upgrades || []}
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onConfirm={() => performUpgrade()}
      />
    </>
  );
};

export default NamespaceDetails;
