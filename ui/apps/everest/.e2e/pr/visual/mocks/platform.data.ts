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

// Deterministic fixtures for the platform/auth endpoints so the visual suite is
// fully hermetic (no backend needed). Captured from a live OpenEverest v2 dev
// cluster and stripped of volatile data.
//
// FAKE_JWT is an unsigned, decode-only token. The app never verifies the
// signature client-side (jwt-decode only base64-decodes the payload); every API
// call is mocked, so the token is never validated server-side. Claims:
//   iss = 'everest' (must equal EVEREST_JWT_ISSUER, see src/consts.ts),
//   sub = 'admin:'  (username is the substring before ':' -> 'admin'),
//   exp far in the future so no refresh is scheduled.

export const FAKE_JWT =
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJldmVyZXN0Iiwic3ViIjoiYWRtaW46IiwiZXhwIjo5OTk5OTk5OTk5LCJpYXQiOjE3NTIwMDAwMDB9.c2lnbmF0dXJl';

// POST /v1/auth/token response (AuthTokenResponse).
export const mockAuthTokenResponse = {
  access_token: FAKE_JWT,
  token_type: 'Bearer',
  expires_in: 3600,
};

// GET /v1/version
export const mockVersion = {
  fullCommit: '0a6672caca542f888e663ee21e232c8f07e1a0b0',
  projectName: 'Everest API Server',
  version: 'v0.0.0-0a6672cac',
};

// GET /v1/settings (empty oidcConfig -> manual login, no SSO)
export const mockSettings = {
  oidcConfig: {
    clientId: '',
    issuerURL: '',
    scopes: ['openid', 'profile', 'email'],
  },
};

// GET /v1/permissions (RBAC disabled -> all allowed)
export const mockPermissions = {
  enabled: false,
  permissions: null,
};

// GET /v1/cluster-info (base64 annotations stripped)
export const mockClusterInfo = {
  clusterType: 'generic',
  storageClassNames: ['local-path'],
  storageClasses: [
    {
      allowVolumeExpansion: false,
      metadata: {
        name: 'local-path',
      },
    },
  ],
};

// GET /v1/plugins
export const mockPlugins = [];
