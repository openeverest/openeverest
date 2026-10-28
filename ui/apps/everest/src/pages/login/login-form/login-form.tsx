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

import { useContext } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Divider, Stack, Typography } from '@mui/material';
import { FormProvider, SubmitHandler, useForm } from 'react-hook-form';
import { TextInput } from '@percona/ui-lib';
import { AuthContext } from 'contexts/auth';
import { HiddenInput } from 'components/hidden-input';
import { LoginFormType, loginSchema } from 'pages/login/Login.constants';
import { Messages } from 'pages/login/Login.messages';

export const LoginForm = () => {
  const { login, authStatus, isSsoEnabled } = useContext(AuthContext);
  const methods = useForm<LoginFormType>({
    mode: 'onChange',
    defaultValues: { username: '', password: '' },
    resolver: zodResolver(loginSchema),
  });
  const isLoggingIn = authStatus === 'loggingIn';

  const handleLogin: SubmitHandler<LoginFormType> = ({
    username,
    password,
  }) => {
    login('manual', { username, password });
  };

  return (
    <Stack>
      <Typography variant="h5" component="h1" sx={{ mb: 1 }}>
        {Messages.login}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
        {Messages.insertCredentials}
      </Typography>
      <FormProvider {...methods}>
        <form onSubmit={methods.handleSubmit(handleLogin)}>
          <TextInput
            isRequired
            name="username"
            textFieldProps={{
              type: 'text',
              autoFocus: true,
              autoComplete: 'username',
              label: Messages.username,
              fullWidth: true,
              sx: { mb: 2 },
              disabled: isLoggingIn,
            }}
          />
          <HiddenInput
            name="password"
            textFieldProps={{
              label: Messages.password,
              autoComplete: 'current-password',
              fullWidth: true,
              sx: { mb: 3 },
              disabled: isLoggingIn,
            }}
          />
          <Button
            type="submit"
            size="large"
            variant="contained"
            fullWidth
            data-testid="login-button"
            disabled={
              isLoggingIn ||
              methods.formState.isSubmitting ||
              !methods.formState.isValid
            }
          >
            {Messages.login}
          </Button>
        </form>
      </FormProvider>
      {isSsoEnabled && (
        <>
          <Divider sx={{ my: 2 }}>
            <Typography variant="caption" color="text.secondary">
              {Messages.or}
            </Typography>
          </Divider>
          <Button
            size="large"
            variant="outlined"
            fullWidth
            disabled={isLoggingIn}
            onClick={() => login('sso')}
          >
            {Messages.loginWithSso}
          </Button>
        </>
      )}
    </Stack>
  );
};
