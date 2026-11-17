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
} from 'components/class-authentication';
import {
  buildSaveRequest,
  getFormDefaults,
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

// A destination form that owns the url only.
const pmm: MonitoringClass = {
  metadata: { name: 'pmm' },
  spec: {
    controllerName: 'openeverest.io/monitoring-pmm',
    authModes: [
      { name: 'ServiceAccountToken', credentials: { required: ['token'] } },
    ],
  },
};
Reflect.set(pmm.spec, 'uiSchema', {
  destination: {
    components: {
      url: {
        path: 'parameters.url',
        uiType: 'text',
        fieldParams: { label: 'PMM server URL' },
      },
    },
  },
});

const storedPmm: MonitoringDestination = {
  metadata: { name: 'pmm-prod', namespace: 'team-a', resourceVersion: '7' },
  spec: {
    classRef: { name: 'pmm' },
    authentication: {
      mode: 'ServiceAccountToken',
      credentialsSecretRef: { name: 'pmm-credentials' },
    },
  },
};
Reflect.set(storedPmm.spec, 'parameters', {
  url: 'https://pmm.example.com',
  proxy: '',
  labels: { team: 'a' },
});

const ALL_SECRET_PERMISSIONS: SecretNamespacePermissions = {
  canCreate: ['team-a'],
  canRead: ['team-a'],
  canUpdate: ['team-a'],
};

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
        fallbackParameters: {
          interval: '',
          podMonitorLabels: [{ key: 'release', value: 'kube-prom' }],
        },
      },
      monitoringClass: prometheus,
      secretPermissions: ALL_SECRET_PERMISSIONS,
    });

    expect(request.credentialsSecret).toEqual({
      action: 'create',
      name: 'monitoring-destination-prom-credentials',
      labels: {
        'openeverest.io/monitoring-class': 'prometheus',
        'openeverest.io/definition': 'BasicAuth',
      },
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

  it("keeps the parameters the class form doesn't own as saved, empty ones too", () => {
    const defaults = getFormDefaults({
      original: storedPmm,
      namespace: 'team-a',
      classes: [pmm],
    });
    const request = buildSaveRequest({
      values: {
        ...defaults,
        spec: { parameters: { url: 'https://pmm2.example.com' } },
      },
      monitoringClass: pmm,
      original: storedPmm,
      secretPermissions: ALL_SECRET_PERMISSIONS,
    });

    expect(request.destination.spec.parameters).toEqual({
      url: 'https://pmm2.example.com',
      proxy: '',
      labels: { team: 'a' },
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
