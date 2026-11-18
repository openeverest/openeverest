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

import { isSection, Section } from 'components/ui-generator/ui-generator.types';
import { isPlainObject } from 'components/ui-generator/utils/object-path';
import {
  MonitoringClass,
  MonitoringConditionType,
} from 'shared-types/monitoring-destinations.types';
import { ClassContract, toSectionMap } from 'utils/class-contract';

// The class uiSchema groups its forms by what they fill: the destination's
// spec, the Secret of each auth mode and an Instance's monitoring entry.
export interface MonitoringClassUiSchema {
  destination?: Section;
  instance?: Section;
  credentials: Record<string, Section>;
}

// A class nobody claims (Accepted is not True) cannot serve a destination.
export const isMonitoringClassAccepted = (monitoringClass: MonitoringClass) =>
  (monitoringClass.status?.conditions ?? []).some(
    (condition) =>
      condition.type === MonitoringConditionType.Accepted &&
      condition.status === 'True'
  );

// Whether any change to an entry of the class restarts the pods of the
// provider's Instances, or of any provider's when none is given.
export const restartsPods = (
  monitoringClass: MonitoringClass,
  provider?: string
) =>
  (monitoringClass.status?.providers ?? []).some(
    (entry) =>
      (!provider || entry.name === provider) &&
      entry.disruption === 'RollingRestart'
  );

export const getMonitoringClassName = (monitoringClass: MonitoringClass) =>
  monitoringClass.metadata?.name ?? '';

export const getMonitoringClassTitle = (monitoringClass: MonitoringClass) =>
  monitoringClass.spec.displayName || getMonitoringClassName(monitoringClass);

// The generated CRD types keep uiSchema opaque, so each form is narrowed here.
export const getMonitoringClassUiSchema = (
  monitoringClass: MonitoringClass
): MonitoringClassUiSchema => {
  const uiSchema: unknown = monitoringClass.spec.uiSchema;
  if (!isPlainObject(uiSchema)) {
    return { credentials: {} };
  }
  return {
    destination: isSection(uiSchema.destination)
      ? uiSchema.destination
      : undefined,
    instance: isSection(uiSchema.instance) ? uiSchema.instance : undefined,
    credentials: toSectionMap(uiSchema.credentials),
  };
};

// The label the Secrets API ties a destination's credentials Secret to its
// class by.
export const MONITORING_CLASS_SECRET_LABEL = 'openeverest.io/monitoring-class';

export const toMonitoringClassContract = (
  monitoringClass: MonitoringClass
): ClassContract => ({
  name: getMonitoringClassName(monitoringClass),
  title: getMonitoringClassTitle(monitoringClass),
  description: monitoringClass.spec.description,
  authModes: monitoringClass.spec.authModes,
  credentialsForms: getMonitoringClassUiSchema(monitoringClass).credentials,
  credentialsSecretLabel: MONITORING_CLASS_SECRET_LABEL,
  credentialsSecretFilter: {
    monitoringClass: getMonitoringClassName(monitoringClass),
  },
});
