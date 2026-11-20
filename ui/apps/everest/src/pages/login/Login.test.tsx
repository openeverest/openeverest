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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { enqueueSnackbar } from 'notistack';
import { Route, Routes } from 'react-router-dom';
import { AuthContext } from 'contexts/auth';
import { AuthContextProps } from 'contexts/auth/auth.context.types';
import { SSO_LOGIN_ERROR_KEY } from 'contexts/auth/auth.utils';
import { TestWrapper, resizeScreenSize } from 'utils/test';
import Login from './Login';
import { Messages } from './Login.messages';

vi.mock('notistack', async (importOriginal) => ({
  ...(await importOriginal<typeof import('notistack')>()),
  enqueueSnackbar: vi.fn(),
}));

const renderLogin = (auth: Partial<AuthContextProps> = {}) => {
  const value: AuthContextProps = {
    login: vi.fn(),
    logout: vi.fn(),
    setRedirectRoute: vi.fn(),
    authStatus: 'loggedOut',
    redirectRoute: null,
    isSsoEnabled: false,
    ...auth,
  };

  render(
    <TestWrapper>
      <AuthContext.Provider value={value}>
        <Routes>
          <Route path="/" element={<Login />} />
          <Route path="/databases" element={<div>Databases page</div>} />
        </Routes>
      </AuthContext.Provider>
    </TestWrapper>
  );

  return value;
};

describe('Login', () => {
  beforeEach(() => {
    resizeScreenSize(1200);
    vi.mocked(enqueueSnackbar).mockClear();
    sessionStorage.clear();
  });

  it.each([
    { width: 1200, isSplitScreen: true },
    { width: 900, isSplitScreen: true },
    { width: 800, isSplitScreen: false },
    { width: 400, isSplitScreen: false },
  ])(
    'at $width px shows the split screen: $isSplitScreen',
    ({ width, isSplitScreen }) => {
      resizeScreenSize(width);
      renderLogin();

      expect(!!screen.queryByTestId('login-brand-panel')).toBe(isSplitScreen);
      expect(!!screen.queryByTestId('login-single-column-logo')).toBe(
        !isSplitScreen
      );
      // The links stay reachable in both layouts
      expect(screen.getAllByTestId('login-links')).toHaveLength(1);
    }
  );

  it('submits the entered credentials', async () => {
    const { login } = renderLogin();

    fireEvent.change(screen.getByTestId('text-input-username'), {
      target: { value: 'admin' },
    });
    fireEvent.change(screen.getByTestId('text-input-password'), {
      target: { value: 'secret' },
    });
    fireEvent.click(screen.getByTestId('login-button'));

    await waitFor(() =>
      expect(login).toHaveBeenCalledExactlyOnceWith('manual', {
        username: 'admin',
        password: 'secret',
      })
    );
  });

  it('keeps the submit button enabled and flags empty fields on submit', async () => {
    const { login } = renderLogin();

    const loginButton = screen.getByTestId('login-button');
    expect(loginButton).toBeEnabled();
    fireEvent.click(loginButton);

    expect(await screen.findAllByText('Required')).toHaveLength(2);
    expect(login).not.toHaveBeenCalled();
  });

  it('disables the inputs and both buttons while logging in', () => {
    renderLogin({ authStatus: 'loggingIn', isSsoEnabled: true });

    expect(screen.getByTestId('text-input-username')).toBeDisabled();
    expect(screen.getByTestId('text-input-password')).toBeDisabled();
    expect(screen.getByTestId('login-button')).toBeDisabled();
    expect(
      screen.getByRole('button', { name: Messages.loginWithSso })
    ).toBeDisabled();
  });

  it('redirects to the stored route once logged in', () => {
    renderLogin({ authStatus: 'loggedIn', redirectRoute: '/databases' });

    expect(screen.getByText('Databases page')).toBeInTheDocument();
  });

  it('surfaces an SSO login error once and clears it', () => {
    sessionStorage.setItem(SSO_LOGIN_ERROR_KEY, 'true');
    renderLogin();

    expect(enqueueSnackbar).toHaveBeenCalledExactlyOnceWith(
      Messages.ssoLoginError,
      { variant: 'error' }
    );
    expect(sessionStorage.getItem(SSO_LOGIN_ERROR_KEY)).toBeNull();
  });

  it.each([true, false])(
    'offers SSO only when it is enabled (enabled: %s)',
    (isSsoEnabled) => {
      const { login } = renderLogin({ isSsoEnabled });
      const ssoButtonQuery = {
        name: Messages.loginWithSso,
      };

      if (isSsoEnabled) {
        fireEvent.click(screen.getByRole('button', ssoButtonQuery));
        expect(login).toHaveBeenCalledExactlyOnceWith('sso');
      } else {
        expect(
          screen.queryByRole('button', ssoButtonQuery)
        ).not.toBeInTheDocument();
      }
    }
  );
});
