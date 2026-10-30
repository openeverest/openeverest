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

import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import {
  buildSaveRequest,
  getCredentialsSecretAction,
  getFormDefaults,
} from './monitoring-destination-modal.utils';

// No uiSchema: credentials from the auth mode, parameters from the schema.
const prometheus: MonitoringClass = {
  metadata: { name: 'prometheus' },
  spec: {
    authModes: [
      { name: 'Anonymous' },
      {
        name: 'BasicAuth',
        credentials: { required: ['username', 'password'] },
      },
    ],
    destinationParametersSchema: {
      openAPIV3Schema: {
        type: 'object',
        properties: {
          interval: { type: 'string' },
          podMonitorLabels: {
            type: 'object',
            additionalProperties: { type: 'string' },
          },
          relabelings: { type: 'array' },
        },
      },
    },
  },
};

const stored: MonitoringDestination = {
  apiVersion: 'monitoring.openeverest.io/v1alpha1',
  kind: 'MonitoringDestination',
  metadata: { name: 'prom', namespace: 'team-a', resourceVersion: '42' },
  spec: {
    classRef: { name: 'prometheus' },
    authentication: {
      mode: 'BasicAuth',
      credentialsSecretRef: { name: 'prom-basic' },
    },
    tls: { insecureSkipVerify: true },
  },
};
Reflect.set(stored.spec, 'parameters', {
  interval: '30s',
  relabelings: [{ action: 'keep' }],
});

describe('getCredentialsSecretAction', () => {
  it('creates a Secret for a new Secret-backed destination only', () => {
    const values = { authMode: 'BasicAuth', replaceCredentials: false };
    expect(getCredentialsSecretAction(values, undefined)).toBe('create');
    expect(
      getCredentialsSecretAction(
        { ...values, authMode: 'Anonymous' },
        undefined
      )
    ).toBeUndefined();
  });

  it('keeps the Secret on edit unless replaced or the mode changes', () => {
    const values = { authMode: 'BasicAuth', replaceCredentials: false };
    expect(getCredentialsSecretAction(values, stored)).toBeUndefined();
    expect(
      getCredentialsSecretAction(
        { ...values, replaceCredentials: true },
        stored
      )
    ).toBe('replace');
  });
});

describe('buildSaveRequest', () => {
  it('builds a Secret from the mode keys when the class has no credentials form', () => {
    const request = buildSaveRequest({
      values: {
        ...getFormDefaults({ namespace: 'team-a', classes: [prometheus] }),
        name: 'prom',
        className: 'prometheus',
        authMode: 'BasicAuth',
        credentials: { username: 'u', password: 'p' },
        fallbackParameters: {
          interval: '',
          podMonitorLabels: [{ key: 'release', value: 'kube-prom' }],
        },
      },
      monitoringClass: prometheus,
    });

    expect(request.credentialsSecret).toEqual({
      action: 'create',
      name: 'prom-credentials',
      stringData: { username: 'u', password: 'p' },
    });
    expect(request.destination).toEqual({
      metadata: { name: 'prom' },
      spec: {
        classRef: { name: 'prometheus' },
        authentication: {
          mode: 'BasicAuth',
          credentialsSecretRef: { name: 'prom-credentials' },
        },
        parameters: { podMonitorLabels: { release: 'kube-prom' } },
      },
    });
  });

  it('updates the stored object, keeping what the form does not edit', () => {
    const defaults = getFormDefaults({
      original: stored,
      namespace: 'team-a',
      classes: [prometheus],
    });
    const request = buildSaveRequest({
      values: {
        ...defaults,
        insecureSkipVerify: false,
        caSecretName: 'prom-ca',
        caSecretKey: 'ca.crt',
      },
      monitoringClass: prometheus,
      original: stored,
    });

    expect(request.isEdit).toBe(true);
    expect(request.credentialsSecret).toBeUndefined();
    expect(request.destination).toEqual({
      apiVersion: stored.apiVersion,
      kind: stored.kind,
      metadata: stored.metadata,
      spec: {
        classRef: { name: 'prometheus' },
        authentication: {
          mode: 'BasicAuth',
          credentialsSecretRef: { name: 'prom-basic' },
        },
        tls: { ca: { secretKeyRef: { name: 'prom-ca', key: 'ca.crt' } } },
        parameters: { interval: '30s', relabelings: [{ action: 'keep' }] },
      },
    });
  });

  it('replaces the credentials in the Secret the destination references', () => {
    const request = buildSaveRequest({
      values: {
        ...getFormDefaults({
          original: stored,
          namespace: 'team-a',
          classes: [prometheus],
        }),
        replaceCredentials: true,
        credentials: { username: 'u2', password: 'p2' },
      },
      monitoringClass: prometheus,
      original: stored,
    });

    expect(request.credentialsSecret).toEqual({
      action: 'replace',
      name: 'prom-basic',
      stringData: { username: 'u2', password: 'p2' },
    });
  });
});
