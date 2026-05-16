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

import { useMemo } from 'react';
import {
  DbEngine,
  DbEngineStatus,
  OperatorUpgradePendingAction,
} from 'shared-types/dbEngines.types';
import { UseOperatorsUpgradePlanType } from './useDbEngines';

export type OperatorUpgradePhase =
  | { phase: 'idle' }
  | {
      phase: 'operator-upgrading';
      startedAt: string;
      targetVersion: string;
    }
  | {
      phase: 'actions-required';
      actionable: OperatorUpgradePendingAction[];
      resolvedCount: number;
      totalCount: number;
    }
  | { phase: 'error'; stuckClusters: string[] };

export const useOperatorUpgradePhase = (
  dbEngines: DbEngine[],
  operatorsUpgradePlan: UseOperatorsUpgradePlanType | undefined
): OperatorUpgradePhase => {
  return useMemo(() => {
    const upgradingEngine = dbEngines.find(
      (e) => e.status === DbEngineStatus.UPGRADING
    );
    if (upgradingEngine?.operatorUpgrade) {
      return {
        phase: 'operator-upgrading',
        startedAt: upgradingEngine.operatorUpgrade.startedAt,
        targetVersion: upgradingEngine.operatorUpgrade.targetVersion,
      };
    }

    const pendingActions = operatorsUpgradePlan?.pendingActions ?? [];
    const stuck = pendingActions.filter((a) => a.pendingTask === 'notReady');
    if (stuck.length > 0) {
      return { phase: 'error', stuckClusters: stuck.map((a) => a.name) };
    }

    const actionable = pendingActions.filter(
      (a) => a.pendingTask === 'restart' || a.pendingTask === 'upgradeEngine'
    );
    if (actionable.length > 0) {
      const totalCount = actionable.length;
      const resolvedCount = pendingActions.filter(
        (a) => a.pendingTask === 'ready'
      ).length;
      return {
        phase: 'actions-required',
        actionable,
        resolvedCount,
        totalCount,
      };
    }

    return { phase: 'idle' };
  }, [dbEngines, operatorsUpgradePlan]);
};
