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

import { BoxProps } from '@mui/material';
import { OperatorUpgradePhase } from 'hooks/api/db-engines/useOperatorUpgradePhase';
import { UseOperatorsUpgradePlanType } from 'hooks/api/db-engines';
import {
  DbEngine,
  OperatorUpgradePendingAction,
  OperatorUpgradeTask,
} from 'shared-types/dbEngines.types';

export type UpgradeHeaderProps = {
  upgradeAvailable: boolean;
  pendingUpgradeTasks: boolean;
  upgradeAllowed: boolean;
  onUpgrade: () => void;
  upgradePhase: OperatorUpgradePhase;
} & BoxProps;

export type ClusterStatusTableProps = {
  namespace: string;
  pendingActions: OperatorUpgradePendingAction[];
  dbEngines: DbEngine[];
  upgradePhase: OperatorUpgradePhase;
};

export type UpgradeModalProps = {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  namespace: string;
  operatorsUpgradeTasks: OperatorUpgradeTask[];
};

export type OperatorVersionsHeaderProps = {
  operatorsUpgradePlan: UseOperatorsUpgradePlanType;
};
