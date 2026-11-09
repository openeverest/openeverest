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

import { MonitoringClass } from 'shared-types/monitoring-destinations.types';
import { getMonitoringClassUiSchema } from './monitoring-classes';

const withUiSchema = (uiSchema: unknown): MonitoringClass => {
  const monitoringClass: MonitoringClass = {
    metadata: { name: 'pmm' },
    spec: { authModes: [{ name: 'ServiceAccountToken' }] },
  };
  Reflect.set(monitoringClass.spec, 'uiSchema', uiSchema);
  return monitoringClass;
};

const section = { components: { url: { path: 'parameters.url' } } };

describe('getMonitoringClassUiSchema', () => {
  it('returns the destination, instance and per-mode credentials forms', () => {
    expect(
      getMonitoringClassUiSchema(
        withUiSchema({
          destination: section,
          instance: section,
          credentials: { ServiceAccountToken: section, Broken: 'nope' },
        })
      )
    ).toEqual({
      destination: section,
      instance: section,
      credentials: { ServiceAccountToken: section },
    });
  });

  it('has no forms when the class has no uiSchema', () => {
    expect(getMonitoringClassUiSchema(withUiSchema(undefined))).toEqual({
      credentials: {},
    });
    expect(
      getMonitoringClassUiSchema(withUiSchema({ destination: { label: 'x' } }))
    ).toEqual({ destination: undefined, instance: undefined, credentials: {} });
  });
});
