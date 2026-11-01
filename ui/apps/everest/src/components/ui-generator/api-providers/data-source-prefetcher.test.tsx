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

import { act, render, waitFor } from '@testing-library/react';
import { FormProvider, useForm, type UseFormReturn } from 'react-hook-form';
import { describe, expect, it, vi } from 'vitest';
import {
  ComponentGroup,
  FieldType,
  GroupType,
  Section,
} from 'components/ui-generator/ui-generator.types';
import { TOGGLEABLE_SWITCHES_KEY } from '../utils/toggleable/toggleable';
import { preprocessSchema } from '../utils/preprocess/preprocess-schema';
import { providerRegistry } from './registry';
import type { ProviderOptions } from './types';
import { DataSourcePrefetcher } from './data-source-prefetcher';

vi.mock('hooks/api/useClusterName', () => ({
  useClusterName: () => 'main',
}));

const PROVIDER = 'prefetchTestConfigs';
const SWITCH = `${TOGGLEABLE_SWITCHES_KEY}.advanced~monitoring`;

const { useOptions } = vi.hoisted(() => {
  const loaded: ProviderOptions = {
    options: [{ label: 'pmm', value: 'pmm' }],
    isLoading: false,
    error: null,
    isEmpty: false,
  };
  return { useOptions: vi.fn(() => loaded) };
});

providerRegistry.register(PROVIDER, {
  description: 'Test provider',
  useOptions,
  emptyStateFallback: null,
});

const monitoringGroup: ComponentGroup = {
  uiType: 'group',
  groupType: GroupType.Toggleable,
  components: {
    configName: {
      uiType: FieldType.Select,
      path: 'spec.monitoring.configName',
      dataSource: { provider: PROVIDER },
      fieldParams: { label: 'Config' },
    },
  },
};

const sections: Record<string, Section> = preprocessSchema({
  replicaSet: {
    sections: { advanced: { components: { monitoring: monitoringGroup } } },
  },
}).replicaSet.sections;

const sharedProviderSections: Record<string, Section> = preprocessSchema({
  replicaSet: {
    sections: {
      advanced: {
        components: {
          monitoring: monitoringGroup,
          storageConfig: {
            uiType: FieldType.Select,
            path: 'spec.storage.configName',
            dataSource: { provider: PROVIDER },
            fieldParams: { label: 'Storage config' },
          },
        },
      },
    },
  },
}).replicaSet.sections;

const renderPrefetcher = (
  switchOn: boolean,
  schemaSections: Record<string, Section> = sections
) => {
  const formRef: { current?: UseFormReturn } = {};
  const Harness = () => {
    const form = useForm<Record<string, unknown>>({
      defaultValues: {
        [TOGGLEABLE_SWITCHES_KEY]: { 'advanced~monitoring': switchOn },
        spec: { monitoring: { configName: '' }, storage: { configName: '' } },
      },
    });
    formRef.current = form;
    return (
      <FormProvider {...form}>
        <DataSourcePrefetcher sections={schemaSections} namespace="ns" />
      </FormProvider>
    );
  };
  render(<Harness />);
  const form = formRef.current;
  if (!form) throw new Error('form not initialized');
  return form;
};

describe('DataSourcePrefetcher - toggleable groups', () => {
  it('does not fetch a provider used only by a switched-off section', () => {
    useOptions.mockClear();
    renderPrefetcher(false);

    expect(useOptions).not.toHaveBeenCalled();
  });

  it('fetches and preselects once the section is switched on', async () => {
    useOptions.mockClear();
    const form = renderPrefetcher(false);

    act(() => form.setValue(SWITCH, true));

    await waitFor(() =>
      expect(form.getValues('spec.monitoring.configName')).toBe('pmm')
    );
    expect(useOptions).toHaveBeenCalledWith({
      namespace: 'ns',
      cluster: 'main',
    });
  });

  it('preselects only fields outside a switched-off section when a provider is shared', async () => {
    const form = renderPrefetcher(false, sharedProviderSections);

    await waitFor(() =>
      expect(form.getValues('spec.storage.configName')).toBe('pmm')
    );
    expect(form.getValues('spec.monitoring.configName')).toBe('');
  });
});
