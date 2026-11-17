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

import { ReactNode } from 'react';
import { renderHook } from '@testing-library/react';
import { FormProvider, useForm, UseFormReturn } from 'react-hook-form';
import {
  AuthenticationFormValues,
  CredentialsSource,
} from '../class-authentication.types';
import {
  CredentialsScope,
  useResetStaleCredentials,
} from './use-reset-stale-credentials';

// An existing Secret picked, and credentials typed, for s3's AccessKey mode
// in team-a.
const PICKED_AND_TYPED: AuthenticationFormValues = {
  authMode: 'AccessKey',
  credentialsSource: CredentialsSource.Existing,
  credentialsSecretName: '',
  existingSecretName: 's3-credentials',
  credentials: { AWS_ACCESS_KEY_ID: 'AKIA' },
};
const S3_ACCESS_KEY: CredentialsScope = {
  namespace: 'team-a',
  className: 's3',
  authMode: 'AccessKey',
};

const renderScoped = () => {
  let form: UseFormReturn<AuthenticationFormValues> | undefined;
  const Wrapper = ({ children }: { children: ReactNode }) => {
    form = useForm<AuthenticationFormValues>({
      defaultValues: PICKED_AND_TYPED,
    });
    return <FormProvider {...form}>{children}</FormProvider>;
  };
  const { rerender } = renderHook(
    (scope: CredentialsScope) => useResetStaleCredentials(scope),
    { initialProps: S3_ACCESS_KEY, wrapper: Wrapper }
  );
  return { rerender, values: () => form?.getValues() };
};

describe('useResetStaleCredentials', () => {
  it('drops only the existing Secret picked in another namespace', () => {
    const { rerender, values } = renderScoped();

    rerender({ ...S3_ACCESS_KEY });
    expect(values()).toEqual(PICKED_AND_TYPED);

    rerender({ ...S3_ACCESS_KEY, namespace: 'team-b' });
    expect(values()).toEqual({ ...PICKED_AND_TYPED, existingSecretName: '' });
  });

  it.each([
    ['class', { ...S3_ACCESS_KEY, className: 'ceph' }],
    ['mode', { ...S3_ACCESS_KEY, authMode: 'ServiceAccountKey' }],
  ])(
    'drops the existing Secret and the typed credentials once the %s changes',
    (_change, scope) => {
      const { rerender, values } = renderScoped();

      rerender(scope);

      expect(values()).toEqual({
        ...PICKED_AND_TYPED,
        existingSecretName: '',
        credentials: {},
      });
    }
  );
});
