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

import { describe, it, expect } from 'vitest';
import { DbType } from '@percona/types';
import { DbWizardFormFields } from 'consts';
import { resourcesFormSchema, ResourceSize } from './constants';
import { Messages } from './messages';

const schema = resourcesFormSchema({}, true, true, true);

const baseInput = {
  dbType: DbType.Mysql,
  sharding: false,
  [DbWizardFormFields.cpu]: 1,
  [DbWizardFormFields.memory]: 1,
  [DbWizardFormFields.disk]: 25,
  [DbWizardFormFields.diskUnit]: 'Gi',
  [DbWizardFormFields.resourceSizePerNode]: ResourceSize.custom,
  [DbWizardFormFields.numberOfNodes]: '1',
  [DbWizardFormFields.proxyCpu]: 1,
  [DbWizardFormFields.proxyMemory]: 1,
  [DbWizardFormFields.resourceSizePerProxy]: ResourceSize.custom,
  [DbWizardFormFields.numberOfProxies]: '1',
  [DbWizardFormFields.nodeRequestsSynced]: true,
  [DbWizardFormFields.proxyRequestsSynced]: true,
};

const findIssue = (
  result: ReturnType<typeof schema.safeParse>,
  field: string
) => {
  if (result.success) {
    return undefined;
  }
  return result.error.issues.find((issue) => issue.path[0] === field);
};

describe('resourcesFormSchema – requests required when unsynced', () => {
  it('allows empty requests when requests are synced with limits', () => {
    const result = schema.safeParse({
      ...baseInput,
      [DbWizardFormFields.nodeRequestsSynced]: true,
      [DbWizardFormFields.proxyRequestsSynced]: true,
      [DbWizardFormFields.cpuRequests]: '',
      [DbWizardFormFields.memoryRequests]: '',
      [DbWizardFormFields.proxyCpuRequests]: '',
      [DbWizardFormFields.proxyMemoryRequests]: '',
    });
    expect(result.success).toBe(true);
  });

  it('requires node CPU/Memory requests when node requests are unsynced', () => {
    const result = schema.safeParse({
      ...baseInput,
      [DbWizardFormFields.nodeRequestsSynced]: false,
      [DbWizardFormFields.cpuRequests]: '',
      [DbWizardFormFields.memoryRequests]: '',
    });
    expect(result.success).toBe(false);
    expect(findIssue(result, DbWizardFormFields.cpuRequests)?.message).toBe(
      Messages.requestRequired
    );
    expect(findIssue(result, DbWizardFormFields.memoryRequests)?.message).toBe(
      Messages.requestRequired
    );
  });

  it('requires proxy CPU/Memory requests when proxy requests are unsynced', () => {
    const result = schema.safeParse({
      ...baseInput,
      [DbWizardFormFields.proxyRequestsSynced]: false,
      [DbWizardFormFields.proxyCpuRequests]: '',
      [DbWizardFormFields.proxyMemoryRequests]: '',
    });
    expect(result.success).toBe(false);
    expect(
      findIssue(result, DbWizardFormFields.proxyCpuRequests)?.message
    ).toBe(Messages.requestRequired);
    expect(
      findIssue(result, DbWizardFormFields.proxyMemoryRequests)?.message
    ).toBe(Messages.requestRequired);
  });

  it('accepts valid requests below the limits when unsynced', () => {
    const result = schema.safeParse({
      ...baseInput,
      [DbWizardFormFields.nodeRequestsSynced]: false,
      [DbWizardFormFields.proxyRequestsSynced]: false,
      [DbWizardFormFields.cpuRequests]: 0.5,
      [DbWizardFormFields.memoryRequests]: 0.5,
      [DbWizardFormFields.proxyCpuRequests]: 0.5,
      [DbWizardFormFields.proxyMemoryRequests]: 0.5,
    });
    expect(result.success).toBe(true);
  });

  it('still rejects requests above the limits when unsynced', () => {
    const result = schema.safeParse({
      ...baseInput,
      [DbWizardFormFields.nodeRequestsSynced]: false,
      [DbWizardFormFields.cpuRequests]: 2,
      [DbWizardFormFields.memoryRequests]: 2,
    });
    expect(result.success).toBe(false);
    expect(findIssue(result, DbWizardFormFields.cpuRequests)?.message).toBe(
      'CPU request cannot exceed CPU limit'
    );
    expect(findIssue(result, DbWizardFormFields.memoryRequests)?.message).toBe(
      'Memory request cannot exceed Memory limit'
    );
  });
});
