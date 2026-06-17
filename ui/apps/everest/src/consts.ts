// everest
// Copyright (C) 2023 Percona LLC
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

import { ImportFields } from 'components/cluster-form/import/import.types';

export const IP_REGEX =
  /^([0-9]{1,3}\.){3}[0-9]{1,3}(\/([0-9]|[1-2][0-9]|3[0-2]))?$/;

export const LOAD_BALANCER_ANNOTATION_REGEX =
  /^(?:(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)*[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\/)?([a-z0-9](?:[a-z0-9_.-]{0,61}[a-z0-9])?)$/;

export const INVALID_SOURCE_RANGE_ERROR = 'Invalid IP address range';
export const DUPLICATE_SOURCE_RANGE_ERROR =
  'Duplicate entry. This IP and netmask combination already exists.';
export const FILE_NOT_INSTANCE_OF_FILE_ERROR = 'Not a valid file';
export const SOURCE_RANGE_PLACEHOLDER =
  'Enter IP with netmask (e.g. 192.168.1.1/24)';
export const DATE_FORMAT = "dd'/'MM'/'yyyy 'at' HH':'mm";
export const PITR_DATE_FORMAT = "dd'/'MM'/'yyyy 'at' HH':'mm':'ss";

export const MAX_DB_CLUSTER_NAME_LENGTH = 22;
// export const MAX_RFC_1123_NAME_LENGTH = 63;
export const MAX_SCHEDULE_NAME_LENGTH = 57;
export const EVEREST_JWT_ISSUER = 'everest';
export const PG_SLOTS_LIMIT = 3;
export const EVEREST_READ_ONLY_FINALIZER =
  'everest.percona.com/readonly-protection';
export const EVEREST_POLICY_IN_USE_FINALIZER =
  'everest.percona.com/in-use-protection';

export enum DbWizardForm {
  dbName = 'dbName',
  provider = 'provider',
  k8sNamespace = 'k8sNamespace',
  topology = 'topology.type',
}

export const DbWizardFormFields = {
  ...ImportFields,
  ...DbWizardForm,
};

export const EKS_DEFAULT_LOAD_BALANCER_CONFIG = 'eks-default';

export const EMPTY_LOAD_BALANCER_CONFIGURATION = '- No configuration -';
