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

import { render, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { FormProvider, useForm, type UseFormReturn } from 'react-hook-form';
import { api } from 'api/api';
import { Component, FieldType, Section } from '../ui-generator.types';
import { SecretNamespaceSync } from './secret-namespace-sync';

vi.mock('api/api', () => ({ api: { get: vi.fn() } }));

vi.mock('hooks/api/useClusterName', () => ({
  useClusterName: () => 'main',
}));

const secretField = (path: string, readOnly = false): Component => ({
  uiType: FieldType.Secret,
  path,
  fieldParams: { label: path, definition: 'user', readOnly },
});

const sections: Record<string, Section> = {
  users: {
    components: {
      kept: secretField('spec.kept'),
      locked: secretField('spec.locked', true),
      cleared: secretField('spec.cleared'),
    },
  },
};

const renderSync = (
  values: Record<string, unknown>,
  namespace: string | undefined
) => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const formRef: { current?: UseFormReturn } = {};
  const Harness = ({ namespace }: { namespace?: string }) => {
    const form = useForm({ defaultValues: values });
    formRef.current = form;
    return (
      <QueryClientProvider client={queryClient}>
        <FormProvider {...form}>
          <SecretNamespaceSync
            sections={sections}
            namespace={namespace}
            provider="psmdb"
          />
        </FormProvider>
      </QueryClientProvider>
    );
  };
  const { rerender } = render(<Harness namespace={namespace} />);
  const form = formRef.current;
  if (!form) throw new Error('form not initialized');
  return {
    form,
    switchTo: (next: string) => rerender(<Harness namespace={next} />),
  };
};

describe('SecretNamespaceSync', () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset();
  });

  it('clears picks the new namespace lacks, keeping the rest', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        items: [
          { metadata: { name: 'creds-a' } },
          {
            metadata: {
              name: 'creds-x',
              deletionTimestamp: '2026-10-12T00:00:00Z',
            },
          },
        ],
      },
    });
    const { form, switchTo } = renderSync(
      { spec: { kept: 'creds-a', locked: 'creds-x', cleared: 'creds-x' } },
      'ns-1'
    );

    switchTo('ns-2');

    await waitFor(() => expect(form.getValues('spec.cleared')).toBe(''));
    expect(form.getValues('spec.kept')).toBe('creds-a');
    expect(form.getValues('spec.locked')).toBe('creds-x');
    expect(api.get).toHaveBeenCalledWith(
      'clusters/main/namespaces/ns-2/secrets',
      { params: { provider: 'psmdb', definition: 'user' } }
    );
  });

  it('leaves picks unchecked when the first namespace is set', () => {
    const { form, switchTo } = renderSync(
      { spec: { cleared: 'creds-x' } },
      undefined
    );

    switchTo('ns-1');

    expect(api.get).not.toHaveBeenCalled();
    expect(form.getValues('spec.cleared')).toBe('creds-x');
  });
});
