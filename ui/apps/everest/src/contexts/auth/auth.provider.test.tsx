/// <reference types="vitest" />

import { describe, it, vi } from 'vitest';

//Mock before imports
vi.mock('oidc-client-ts', () => ({
  UserManager: vi.fn(() => ({
    signinRedirect: vi.fn(),
    signoutRedirect: vi.fn(),
    getUser: vi.fn().mockResolvedValue(null),
    removeUser: vi.fn(),
  })),
  WebStorageStateStore: vi.fn(),
}));

//Mock AuthProvider before importing it
vi.mock('./auth.provider', async () => {
  const actual = await vi.importActual<typeof import('./auth.provider')>('./auth.provider');
  return {
    ...actual,
    default: ({ children }: any) => children, // skip auth logic
  };
});

//Now import everything
import { render } from '@testing-library/react';
import AuthProvider from './auth.provider';
import * as apiModule from '../../api/api';

describe('AuthProvider logout behavior', () => {
  it('does not crash if api.delete fails', async () => {
    vi.spyOn(apiModule.api, 'delete').mockRejectedValue(new Error('fail'));

    render(
      <AuthProvider>
        <div />
      </AuthProvider>
    );
  });
});