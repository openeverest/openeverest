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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { api } from 'api/api';
import { buildMonitoringParametersSchema } from 'components/monitoring-destinations-picker/monitoring-destinations-picker.utils';
import { useFormEngine } from 'components/ui-generator/form-engine/use-form-engine';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import {
  FieldType,
  FormMode,
  TextValidation,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { preprocessSchema } from 'components/ui-generator/utils/preprocess/preprocess-schema';
import { useCreateDbInstance } from 'hooks/api/db-instances/useCreateDbInstance';
import { resizeScreenSize, TestWrapper } from 'utils/test';
import { getDBWizardSchema } from './database-form-schema';

interface SubmissionHarnessProps {
  schema: TopologyUISchemas;
}

const SubmissionHarness = ({ schema }: SubmissionHarnessProps) => {
  const engine = useFormEngine({
    uiSchema: schema,
    selectedTopology: 'single',
    formMode: FormMode.New,
  });
  const methods = useForm<Record<string, unknown>>({
    mode: 'onChange',
    resolver: zodResolver(
      getDBWizardSchema(
        [],
        engine.zodSchema,
        buildMonitoringParametersSchema(new Map())
      ),
      undefined,
      { mode: 'sync' }
    ),
    defaultValues: {
      provider: 'test-provider',
      dbName: 'transform-instance',
      k8sNamespace: 'default',
      topology: { type: 'single' },
      spec: { zone: '' },
    },
  });
  const { mutate } = useCreateDbInstance();
  return (
    <FormProvider {...methods}>
      <form
        onSubmit={methods.handleSubmit((data) =>
          mutate({ formValue: engine.postprocess(data) })
        )}
      >
        <UIGenerator sections={engine.sections} sectionKey="instance" />
        <button type="submit">Create instance</button>
      </form>
    </FormProvider>
  );
};

const TRANSFORM_CASES: {
  name: string;
  validation: TextValidation;
  raw: string;
  expected: string;
}[] = [
  {
    name: 'trim',
    validation: { trim: true },
    raw: ' EU-West ',
    expected: 'EU-West',
  },
  {
    name: 'lowercase',
    validation: { toLowerCase: true },
    raw: 'EU-West',
    expected: 'eu-west',
  },
  {
    name: 'uppercase',
    validation: { toUpperCase: true },
    raw: 'Eu-West',
    expected: 'EU-WEST',
  },
  {
    name: 'trim and lowercase',
    validation: { trim: true, toLowerCase: true },
    raw: ' EU-West ',
    expected: 'eu-west',
  },
  {
    name: 'trim and uppercase',
    validation: { trim: true, toUpperCase: true },
    raw: ' EU-West ',
    expected: 'EU-WEST',
  },
  {
    name: 'trim, lowercase, then uppercase',
    validation: { trim: true, toLowerCase: true, toUpperCase: true },
    raw: ' EU-West ',
    expected: 'EU-WEST',
  },
  {
    name: 'no transforms',
    validation: {},
    raw: ' EU-West ',
    expected: ' EU-West ',
  },
];

const renderSubmission = (schema: TopologyUISchemas) => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({ data: {} });
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  });
  render(
    <TestWrapper>
      <QueryClientProvider client={queryClient}>
        <SubmissionHarness schema={schema} />
      </QueryClientProvider>
    </TestWrapper>
  );
  return post;
};

describe('instance creation with UI Generator text transforms', () => {
  beforeEach(() => resizeScreenSize(1200));
  afterEach(() => vi.restoreAllMocks());

  it.each(TRANSFORM_CASES)(
    'submits $name to the Instance API while keeping the typed text visible',
    async ({ validation, raw, expected }) => {
      const schema = preprocessSchema(
        {
          single: {
            sections: {
              instance: {
                components: {
                  zone: {
                    uiType: FieldType.Text,
                    path: 'spec.zone',
                    fieldParams: { label: 'Zone' },
                    validation,
                  },
                },
              },
            },
          },
        },
        undefined,
        FormMode.New
      );
      const post = renderSubmission(schema);
      const input = screen.getByLabelText('Zone');
      fireEvent.change(input, { target: { value: raw } });
      expect(input).toHaveValue(raw);
      fireEvent.click(screen.getByRole('button', { name: 'Create instance' }));
      await waitFor(() => expect(post).toHaveBeenCalledOnce());
      expect(post).toHaveBeenCalledExactlyOnceWith(
        'clusters/main/namespaces/default/instances',
        {
          apiVersion: 'core.openeverest.io/v1alpha1',
          kind: 'Instance',
          metadata: { name: 'transform-instance' },
          spec: {
            providerRef: { name: 'test-provider' },
            topology: { type: 'single' },
            zone: expected,
          },
        }
      );
      expect(input).toHaveValue(raw);
    }
  );

  it('maps transformed text to every target and preserves numeric and quantity processing in the API payload', async () => {
    const schema = preprocessSchema(
      {
        single: {
          sections: {
            instance: {
              components: {
                zone: {
                  uiType: FieldType.Text,
                  path: ['spec.zone', 'spec.proxy.zone'],
                  fieldParams: { label: 'Zone' },
                  validation: { trim: true, toLowerCase: true },
                },
                replicas: {
                  uiType: FieldType.Number,
                  path: 'spec.replicas',
                  fieldParams: { label: 'Replicas' },
                  validation: { required: true },
                },
                storage: {
                  uiType: FieldType.Quantity,
                  path: 'spec.storage',
                  fieldParams: { label: 'Storage', unit: 'Gi' },
                  validation: { required: true },
                },
              },
            },
          },
        },
      },
      undefined,
      FormMode.New
    );
    const post = renderSubmission(schema);
    fireEvent.change(screen.getByLabelText('Zone'), {
      target: { value: ' EU-West ' },
    });
    fireEvent.change(screen.getByLabelText(/Replicas/), {
      target: { value: '3' },
    });
    fireEvent.change(screen.getByLabelText(/Storage/), {
      target: { value: '2' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Create instance' }));
    await waitFor(() => expect(post).toHaveBeenCalledOnce());
    expect(post).toHaveBeenCalledWith(
      'clusters/main/namespaces/default/instances',
      expect.objectContaining({
        spec: {
          providerRef: { name: 'test-provider' },
          topology: { type: 'single' },
          zone: 'eu-west',
          proxy: { zone: 'eu-west' },
          replicas: 3,
          storage: '2Gi',
        },
      })
    );
    expect(screen.getByLabelText('Zone')).toHaveValue(' EU-West ');
  });

  it('blocks submission when trimming a required field makes it empty', async () => {
    const schema = preprocessSchema(
      {
        single: {
          sections: {
            instance: {
              components: {
                zone: {
                  uiType: FieldType.Text,
                  path: 'spec.zone',
                  fieldParams: { label: 'Zone' },
                  validation: { trim: true, required: true },
                },
              },
            },
          },
        },
      },
      undefined,
      FormMode.New
    );
    const post = renderSubmission(schema);
    fireEvent.change(screen.getByLabelText(/Zone/), {
      target: { value: '   ' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Create instance' }));
    expect(await screen.findByText('Field is required')).toBeInTheDocument();
    expect(post).not.toHaveBeenCalled();
    expect(screen.getByLabelText(/Zone/)).toHaveValue('   ');
  });
});
