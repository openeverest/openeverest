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

import {
  FieldType,
  FormMode,
  GroupType,
  Section,
} from 'components/ui-generator/ui-generator.types';
import { keyValueRow } from 'components/ui-generator/test-fixtures/key-value-row.fixture';
import { setByPath } from '../object-path';
import { Messages } from '../openapi-constraints/openapi-constraints.messages';
import { getToggleableMeta } from '../toggleable/toggleable';
import {
  getSectionFormDefaults,
  getSectionFormUpdate,
  getSectionFormValues,
  getSectionFormZodSchema,
  toSectionForm,
} from './section-form';
import { SectionForm, SectionFormOptions } from './section-form.types';

const SECTION_KEY = 'destination';

const section: Section = {
  components: {
    url: {
      uiType: FieldType.Text,
      path: 'parameters.url',
      fieldParams: { label: 'URL' },
    },
    interval: {
      uiType: FieldType.Text,
      path: 'parameters.interval',
      fieldParams: { label: 'Interval', badge: 's', badgeToApi: true },
    },
    region: {
      uiType: FieldType.Text,
      path: 'parameters.region',
      fieldParams: { label: 'Region', modes: { edit: { disabled: false } } },
    },
    memory: {
      uiType: FieldType.Quantity,
      path: 'parameters.memory',
      fieldParams: { label: 'Memory', unit: 'Gi' },
    },
    advanced: {
      uiType: 'group',
      groupType: GroupType.Toggleable,
      label: 'Advanced',
      components: {
        timeout: {
          uiType: FieldType.Text,
          path: 'parameters.timeout',
          fieldParams: { label: 'Timeout' },
        },
      },
    },
  },
};

const parametersSchema = {
  type: 'object',
  required: ['url', 'timeout'],
  properties: {
    url: { type: 'string', pattern: '^https?://' },
    interval: { type: 'string', pattern: '^[0-9]+s$' },
    region: { type: 'string', maxLength: 3 },
    timeout: { type: 'string', pattern: '^[0-9]+s$' },
  },
};

const toForm = (options: SectionFormOptions = {}) =>
  toSectionForm(SECTION_KEY, section, {
    contract: { path: 'parameters', schema: parametersSchema },
    ...options,
  });

const advancedSwitch = (form: SectionForm) => {
  const group = form.sections[SECTION_KEY].components.advanced;
  return getToggleableMeta(group)?.switchName ?? '';
};

const withValues = (
  form: SectionForm,
  values: Record<string, unknown>,
  base = getSectionFormDefaults(form)
) => {
  const result = structuredClone(base);
  Object.entries(values).forEach(([path, value]) =>
    setByPath(result, path, value)
  );
  return result;
};

const issuesOf = (form: SectionForm, values: Record<string, unknown>) =>
  (getSectionFormZodSchema(form).safeParse(values).error?.issues ?? []).map(
    ({ path, message }) => ({ path: path.join('.'), message })
  );

describe('section form contract', () => {
  it('checks what the API gets, reporting on the field', () => {
    const form = toForm();

    expect(
      issuesOf(
        form,
        withValues(form, {
          'parameters.url': 'https://pmm.example.com',
          'parameters.interval': '30',
        })
      )
    ).toEqual([]);
    expect(
      issuesOf(
        form,
        withValues(form, {
          'parameters.url': 'pmm.example.com',
          // Sent as "30ms".
          'parameters.interval': '30m',
        })
      )
    ).toEqual([
      { path: 'parameters.url', message: Messages.pattern },
      { path: 'parameters.interval', message: Messages.pattern },
    ]);
  });

  it('asks for a value the schema requires', () => {
    const form = toForm();

    expect(issuesOf(form, withValues(form, {}))).toEqual([
      { path: 'parameters.url', message: Messages.required },
    ]);
  });

  it('leaves a field the user cannot change to the server', () => {
    const values = (form: SectionForm) =>
      withValues(form, {
        'parameters.url': 'https://pmm.example.com',
        'parameters.region': 'eu-west-1',
      });
    const editable = toForm();
    const fixed = toForm({
      overrides: { 'parameters.region': { disabled: true } },
    });

    expect(issuesOf(editable, values(editable))).toEqual([
      { path: 'parameters.region', message: Messages.maxLength(3) },
    ]);
    expect(issuesOf(fixed, values(fixed))).toEqual([]);
  });

  it('checks a toggleable group only while it is switched on, as it then sends nothing', () => {
    const form = toForm();
    const values = (on: boolean) =>
      withValues(form, {
        'parameters.url': 'https://pmm.example.com',
        'parameters.timeout': 'soon',
        [advancedSwitch(form)]: on,
      });

    expect(issuesOf(form, values(false))).toEqual([]);
    expect(issuesOf(form, values(true))).toEqual([
      { path: 'parameters.timeout', message: Messages.pattern },
    ]);
  });
});

describe('section form edit', () => {
  const saved = {
    classRef: { name: 'pmm' },
    parameters: {
      url: 'https://old.example.com',
      interval: '60s',
      memory: '10240Mi',
      timeout: '5s',
      // Keys the form doesn't own, empty ones included.
      custom: '',
      extra: null,
    },
  };

  it('reads saved values the way the fields show them', () => {
    const form = toForm({ formMode: FormMode.Edit });
    const values = getSectionFormValues(form, saved);

    expect(values).toMatchObject({
      parameters: {
        url: 'https://old.example.com',
        interval: '60',
        memory: 10,
        region: '',
      },
    });
    expect(values).toHaveProperty(advancedSwitch(form), true);
  });

  it('sends the saved object with the owned fields replaced and the rest kept', () => {
    const form = toForm({ formMode: FormMode.Edit });
    const values = withValues(
      form,
      {
        'parameters.url': 'https://new.example.com',
        [advancedSwitch(form)]: false,
      },
      getSectionFormValues(form, saved)
    );

    expect(getSectionFormUpdate(form, values, saved)).toEqual({
      classRef: { name: 'pmm' },
      parameters: {
        url: 'https://new.example.com',
        interval: '60s',
        // Untouched, so sent as saved rather than as 10Gi.
        memory: '10240Mi',
        custom: '',
        extra: null,
      },
    });
  });

  it('lets a consumer override win over the schema modes', () => {
    const form = toForm({
      formMode: FormMode.Edit,
      overrides: { 'parameters.region': { disabled: true } },
    });
    const region = form.sections[SECTION_KEY].components.region;

    expect('fieldParams' in region && region.fieldParams).toMatchObject({
      disabled: true,
    });
  });
});

describe('section form with a map field', () => {
  const mapSection: Section = {
    components: {
      labels: {
        uiType: FieldType.Map,
        path: 'parameters.podMonitorLabels',
        fieldParams: { label: 'PodMonitor labels' },
        itemTemplate: keyValueRow(),
      },
    },
  };
  const form = toSectionForm(SECTION_KEY, mapSection, {
    formMode: FormMode.Edit,
  });
  const saved = {
    parameters: {
      podMonitorLabels: { release: 'prometheus', infra: '' },
      // Not in the schema: an edit must not drop it.
      scrapeTimeout: '10s',
    },
  };

  it('reads a saved map as rows and sends it back unchanged', () => {
    const values = getSectionFormValues(form, saved);

    expect(values).toEqual({
      parameters: {
        podMonitorLabels: [
          { key: 'release', value: 'prometheus' },
          { key: 'infra', value: '' },
        ],
      },
    });
    expect(getSectionFormUpdate(form, values, saved)).toEqual(saved);
  });

  it('removes an entry whose row was removed, and the map with its last one', () => {
    const oneRow = {
      parameters: { podMonitorLabels: [{ key: 'infra', value: '' }] },
    };
    const noRows = { parameters: { podMonitorLabels: [] } };

    expect(getSectionFormUpdate(form, oneRow, saved)).toEqual({
      parameters: { podMonitorLabels: { infra: '' }, scrapeTimeout: '10s' },
    });
    expect(getSectionFormUpdate(form, noRows, saved)).toEqual({
      parameters: { scrapeTimeout: '10s' },
    });
  });
});
