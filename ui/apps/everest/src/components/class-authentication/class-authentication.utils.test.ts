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
  CredentialsSource,
  SavedAuthentication,
  SecretNamespacePermissions,
} from './class-authentication.types';
import {
  getCredentialsSource,
  getCredentialsSources,
  getKeptCredentialsSecretName,
} from './class-authentication.utils';

const saved: SavedAuthentication = {
  mode: 'BasicAuth',
  credentialsSecretRef: { name: 'prom-basic' },
};

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
    saved?: SavedAuthentication;
    permissions: SecretNamespacePermissions;
    expected: CredentialsSource[];
  }>([
    {
      name: 'a new object: new or existing Secret',
      authMode: 'BasicAuth',
      permissions: ALL_SECRET_PERMISSIONS,
      expected: [New, Existing],
    },
    {
      name: 'a new object without create: existing only',
      authMode: 'BasicAuth',
      permissions: only('canRead'),
      expected: [Existing],
    },
    {
      name: 'a new object without read: new only',
      authMode: 'BasicAuth',
      permissions: only('canCreate'),
      expected: [New],
    },
    {
      name: 'an edit: keep, update or switch the Secret',
      authMode: 'BasicAuth',
      saved,
      permissions: ALL_SECRET_PERMISSIONS,
      expected: [Keep, Update, Existing],
    },
    {
      name: 'an edit without update: no in-place update',
      authMode: 'BasicAuth',
      saved,
      permissions: only('canRead', 'canCreate'),
      expected: [Keep, Existing],
    },
    {
      name: 'an edit without Secret permissions: keep only',
      authMode: 'BasicAuth',
      saved,
      permissions: only(),
      expected: [Keep],
    },
    {
      name: 'an edit to another mode: its Secret no longer fits',
      authMode: 'Token',
      saved,
      permissions: ALL_SECRET_PERMISSIONS,
      expected: [New, Existing],
    },
    {
      name: 'a mode without a Secret: none',
      authMode: 'Anonymous',
      saved,
      permissions: ALL_SECRET_PERMISSIONS,
      expected: [],
    },
  ])('offers $name', ({ authMode, saved, permissions, expected }) => {
    expect(
      getCredentialsSources(
        { authMode, namespace: 'team-a' },
        saved,
        permissions
      )
    ).toEqual(expected);
  });

  it('falls back to the first offered source once the pick no longer applies', () => {
    expect(
      getCredentialsSource(
        { authMode: 'Token', namespace: 'team-a', credentialsSource: Keep },
        saved,
        ALL_SECRET_PERMISSIONS
      )
    ).toBe(New);
  });
});

describe('getKeptCredentialsSecretName', () => {
  it('keeps the current Secret only for the mode it was made for', () => {
    expect(getKeptCredentialsSecretName('BasicAuth', saved)).toBe('prom-basic');
    expect(getKeptCredentialsSecretName('Anonymous', saved)).toBeUndefined();
    expect(
      getKeptCredentialsSecretName('BasicAuth', undefined)
    ).toBeUndefined();
  });
});
