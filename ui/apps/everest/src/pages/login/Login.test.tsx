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
import { AuthContext } from 'contexts/auth';
import { AuthContextProps } from 'contexts/auth/auth.context.types';
import { TestWrapper, resizeScreenSize } from 'utils/test';
import Login from './Login';

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
        <Login />
      </AuthContext.Provider>
    </TestWrapper>
  );

  return value;
};

describe('Login', () => {
  it('shows the brand panel next to the form on larger screens', () => {
    resizeScreenSize(1200);
    renderLogin();

    expect(screen.getByTestId('login-brand-panel')).toBeInTheDocument();
    expect(screen.queryByTestId('login-mobile-logo')).not.toBeInTheDocument();
    expect(screen.getByTestId('login-button')).toBeInTheDocument();
  });

  it('shows only a logo and the form on mobile', () => {
    resizeScreenSize(400);
    renderLogin();

    expect(screen.queryByTestId('login-brand-panel')).not.toBeInTheDocument();
    expect(screen.getByTestId('login-mobile-logo')).toBeInTheDocument();
    expect(screen.getByTestId('login-button')).toBeInTheDocument();
  });

  it('submits the entered credentials', async () => {
    resizeScreenSize(1200);
    const { login } = renderLogin();

    fireEvent.change(screen.getByTestId('text-input-username'), {
      target: { value: 'admin' },
    });
    fireEvent.change(screen.getByTestId('text-input-password'), {
      target: { value: 'secret' },
    });
    await waitFor(() =>
      expect(screen.getByTestId('login-button')).toBeEnabled()
    );
    fireEvent.click(screen.getByTestId('login-button'));

    await waitFor(() =>
      expect(login).toHaveBeenCalledExactlyOnceWith('manual', {
        username: 'admin',
        password: 'secret',
      })
    );
  });

  it('offers SSO only when it is enabled', () => {
    resizeScreenSize(1200);
    const { login } = renderLogin({ isSsoEnabled: true });

    fireEvent.click(screen.getByRole('button', { name: 'Log in with SSO' }));

    expect(login).toHaveBeenCalledWith('sso');
  });

  it('hides SSO when it is disabled', () => {
    resizeScreenSize(1200);
    renderLogin({ isSsoEnabled: false });

    expect(
      screen.queryByRole('button', { name: 'Log in with SSO' })
    ).not.toBeInTheDocument();
  });
});
