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
  CredentialsSource,
  SecretNamespacePermissions,
} from './monitoring-destination-modal.types';
import {
  buildSaveRequest,
  getCredentialsSource,
  getCredentialsSources,
  getFormDefaults,
  getKeptCredentialsSecretName,
} from './monitoring-destination-modal.utils';

// No uiSchema: credentials from the auth mode, parameters from the schema.
const prometheus: MonitoringClass = {
  metadata: { name: 'prometheus' },
  spec: {
    controllerName: 'openeverest.io/monitoring-prometheus',
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

const ALL_SECRET_PERMISSIONS: SecretNamespacePermissions = {
  canCreate: ['team-a'],
  canRead: ['team-a'],
  canUpdate: ['team-a'],
};
const only = (
  ...granted: (keyof SecretNamespacePermissions)[]
): SecretNamespacePermissions => ({
  canCreate: granted.includes('canCreate') ? ['team-a'] : [],
  canRead: granted.includes('canRead') ? ['team-a'] : [],
  canUpdate: granted.includes('canUpdate') ? ['team-a'] : [],
});

describe('getCredentialsSources', () => {
  const { Keep, Update, New, Existing } = CredentialsSource;

  it.each<{
    name: string;
    authMode: string;
    original?: MonitoringDestination;
    permissions: SecretNamespacePermissions;
    expected: CredentialsSource[];
  }>([
    {
      name: 'a new destination: new or existing Secret',
      authMode: 'BasicAuth',
      permissions: ALL_SECRET_PERMISSIONS,
      expected: [New, Existing],
    },
    {
      name: 'a new destination without create: existing only',
      authMode: 'BasicAuth',
      permissions: only('canRead'),
      expected: [Existing],
    },
    {
      name: 'a new destination without read: new only',
      authMode: 'BasicAuth',
      permissions: only('canCreate'),
      expected: [New],
    },
    {
      name: 'an edit: keep, update or switch the Secret',
      authMode: 'BasicAuth',
      original: stored,
      permissions: ALL_SECRET_PERMISSIONS,
      expected: [Keep, Update, Existing],
    },
    {
      name: 'an edit without update: no in-place update',
      authMode: 'BasicAuth',
      original: stored,
      permissions: only('canRead', 'canCreate'),
      expected: [Keep, Existing],
    },
    {
      name: 'an edit without Secret permissions: keep only',
      authMode: 'BasicAuth',
      original: stored,
      permissions: only(),
      expected: [Keep],
    },
    {
      name: 'an edit to another mode: its Secret no longer fits',
      authMode: 'Token',
      original: stored,
      permissions: ALL_SECRET_PERMISSIONS,
      expected: [New, Existing],
    },
    {
      name: 'a mode without a Secret: none',
      authMode: 'Anonymous',
      original: stored,
      permissions: ALL_SECRET_PERMISSIONS,
      expected: [],
    },
  ])('offers $name', ({ authMode, original, permissions, expected }) => {
    expect(
      getCredentialsSources(
        { authMode, namespace: 'team-a' },
        original,
        permissions
      )
    ).toEqual(expected);
  });

  it('falls back to the first offered source once the pick no longer applies', () => {
    expect(
      getCredentialsSource(
        { authMode: 'Token', namespace: 'team-a', credentialsSource: Keep },
        stored,
        ALL_SECRET_PERMISSIONS
      )
    ).toBe(New);
  });
});

describe('getKeptCredentialsSecretName', () => {
  it('keeps the current Secret only for the mode it was made for', () => {
    expect(getKeptCredentialsSecretName('BasicAuth', stored)).toBe(
      'prom-basic'
    );
    expect(getKeptCredentialsSecretName('Anonymous', stored)).toBeUndefined();
    expect(
      getKeptCredentialsSecretName('BasicAuth', undefined)
    ).toBeUndefined();
  });
});

describe('getFormDefaults', () => {
  const pmm: MonitoringClass = {
    metadata: { name: 'pmm' },
    spec: {
      controllerName: 'openeverest.io/monitoring-pmm',
      authModes: [
        { name: 'ServiceAccountToken', credentials: { required: ['token'] } },
      ],
    },
  };

  it('starts a new destination in the given class, with its only auth mode', () => {
    const defaults = getFormDefaults({
      namespace: 'team-a',
      classes: [pmm],
      className: 'pmm',
    });

    expect(defaults).toMatchObject({
      namespace: 'team-a',
      className: 'pmm',
      authMode: 'ServiceAccountToken',
    });
    expect(defaults.credentials).toHaveProperty('token');
  });

  it('leaves the auth mode to the user when the class has several', () => {
    expect(
      getFormDefaults({
        namespace: 'team-a',
        classes: [prometheus],
        className: 'prometheus',
      })
    ).toMatchObject({ className: 'prometheus', authMode: '' });
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
        spec: {
          parameters: {
            interval: '',
            podMonitorLabels: [{ key: 'release', value: 'kube-prom' }],
          },
        },
      },
      monitoringClass: prometheus,
      secretPermissions: ALL_SECRET_PERMISSIONS,
    });

    expect(request.credentialsSecret).toEqual({
      action: 'create',
      name: 'monitoring-destination-prom-credentials',
      stringData: { username: 'u', password: 'p' },
    });
    expect(request.destination).toEqual({
      metadata: { name: 'prom' },
      spec: {
        classRef: { name: 'prometheus' },
        authentication: {
          mode: 'BasicAuth',
          credentialsSecretRef: {
            name: 'monitoring-destination-prom-credentials',
          },
        },
        parameters: { podMonitorLabels: { release: 'kube-prom' } },
      },
    });
  });

  it('references a picked existing Secret and writes none', () => {
    const request = buildSaveRequest({
      values: {
        ...getFormDefaults({ namespace: 'team-a', classes: [prometheus] }),
        name: 'prom',
        className: 'prometheus',
        authMode: 'BasicAuth',
        credentialsSource: CredentialsSource.Existing,
        existingSecretName: 'prom-credentials',
        credentials: { username: 'u', password: 'p' },
      },
      monitoringClass: prometheus,
      secretPermissions: ALL_SECRET_PERMISSIONS,
    });

    expect(request.credentialsSecret).toBeUndefined();
    expect(request.destination.spec.authentication).toEqual({
      mode: 'BasicAuth',
      credentialsSecretRef: { name: 'prom-credentials' },
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
      secretPermissions: ALL_SECRET_PERMISSIONS,
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

  it('updates the credentials in the Secret the destination already uses', () => {
    const request = buildSaveRequest({
      values: {
        ...getFormDefaults({
          original: stored,
          namespace: 'team-a',
          classes: [prometheus],
        }),
        credentialsSource: CredentialsSource.Update,
        credentials: { username: 'u2', password: 'p2' },
      },
      monitoringClass: prometheus,
      original: stored,
      secretPermissions: ALL_SECRET_PERMISSIONS,
    });

    expect(request.credentialsSecret).toEqual({
      action: 'update',
      name: 'prom-basic',
      stringData: { username: 'u2', password: 'p2' },
    });
    expect(request.destination.spec.authentication).toEqual({
      mode: 'BasicAuth',
      credentialsSecretRef: { name: 'prom-basic' },
    });
  });
});
