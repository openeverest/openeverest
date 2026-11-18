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

import { parse } from 'yaml';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  FormMode,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { Provider } from 'shared-types/api.types';
import { preprocessSchema } from './preprocess-schema';
// Also run by the Go validator (provider-runtime/uischema), so both apply the same rules.
import sharedCasesYaml from '../../../../../../../../provider-runtime/uischema/testdata/select-defaults.yaml?raw';

interface SharedIssue {
  rule: string;
  topology: string;
  field: string;
}

interface SharedCase {
  name: string;
  provider: Provider;
  uiSchema: TopologyUISchemas;
  issues: SharedIssue[];
}

const { cases }: { cases: SharedCase[] } = parse(sharedCasesYaml);

// The UI only warns about the field, not which rule it broke.
const WARNED_FIELD = /^\[UISchema\] "([^"]+)" (?:sets|names) default/;

describe('select default rules shared with the Go validator', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it.each(cases)('$name', ({ provider, uiSchema, issues }) => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const warned = new Set<string>();

    Object.entries(uiSchema).forEach(([topology, topologySchema]) => {
      warn.mockClear();
      preprocessSchema({ [topology]: topologySchema }, provider, FormMode.New);
      warn.mock.calls.forEach(([message]) => {
        const field = WARNED_FIELD.exec(String(message))?.[1];
        if (field) warned.add(`${topology}:${field}`);
      });
    });

    expect([...warned].sort()).toEqual(
      [
        ...new Set(issues.map(({ topology, field }) => `${topology}:${field}`)),
      ].sort()
    );
  });
});
