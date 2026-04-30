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

import { CrdsGen, HttpApi } from '@generated/api-types';

export type GetProviders = CrdsGen.components['schemas']['ProviderList'];
export type Provider = CrdsGen.components['schemas']['Provider'];
export type MonitoringConfig =
  CrdsGen.components['schemas']['MonitoringConfig'];
export type MonitoringConfigList =
  CrdsGen.components['schemas']['MonitoringConfigList'];
export type MonitoringConfigCreateParams =
  HttpApi.components['schemas']['MonitoringConfigCreateParams'];
export type MonitoringConfigUpdateParams =
  HttpApi.components['schemas']['MonitoringConfigUpdateParams'];
export type GetInstances = CrdsGen.components['schemas']['InstanceList'];
export type Instance = CrdsGen.components['schemas']['Instance'];
export type InstanceConnectionDetails =
  CrdsGen.components['schemas']['InstanceConnectionDetails'];
export type PhaseType = NonNullable<Instance['status']>['phase'];

export type CreateDbInstancePayload =
  HttpApi.paths['/clusters/{cluster}/namespaces/{namespace}/instances']['post']['requestBody']['content']['application/json'];
export type GetDbInstanceConnectionPayload =
  HttpApi.paths['/clusters/{cluster}/namespaces/{namespace}/instances/{instance}/connection']['get']['responses']['200']['content']['application/json'];
export type GetDbInstancePayload =
  HttpApi.paths['/clusters/{cluster}/namespaces/{namespace}/instances/{instance}']['get']['responses']['200']['content']['application/json'];
export type GetDbInstancesPayload =
  HttpApi.paths['/clusters/{cluster}/namespaces/{namespace}/instances']['get']['responses']['200']['content']['application/json'];
export type UpdateDbInstancePayload =
  HttpApi.paths['/clusters/{cluster}/namespaces/{namespace}/instances/{instance}']['put']['requestBody']['content']['application/json'];
