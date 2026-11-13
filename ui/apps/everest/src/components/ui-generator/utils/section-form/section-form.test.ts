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

import { describe, expect, it } from 'vitest';
import {
  type Section,
  FieldType,
  FormMode,
  TemplateWidgetType,
  WIDGET_UI_TYPE,
} from '../../ui-generator.types';
import {
  getSectionFormValues,
  postprocessSectionForm,
  toSectionForm,
} from './section-form';

const section: Section = {
  components: {
    interval: {
      uiType: FieldType.Text,
      path: 'parameters.interval',
      fieldParams: { label: 'Interval', defaultValue: '30s' },
    },
    labels: {
      uiType: FieldType.Map,
      path: 'parameters.podMonitorLabels',
      fieldParams: { label: 'PodMonitor labels' },
      itemTemplate: {
        uiType: WIDGET_UI_TYPE,
        widgetType: TemplateWidgetType.KeyValue,
      },
    },
  },
};

describe('section form edit values', () => {
  it('reads saved values into the form shape and writes them back unchanged', () => {
    const form = toSectionForm('destination', section, FormMode.Edit);
    const saved = {
      parameters: {
        podMonitorLabels: { release: 'prometheus', infra: '' },
        scrapeTimeout: '10s',
      },
    };

    const values = getSectionFormValues(form, saved);

    expect(values).toEqual({
      parameters: {
        interval: '30s',
        podMonitorLabels: [
          { key: 'release', value: 'prometheus' },
          { key: 'infra', value: '' },
        ],
        // Not in the schema, but saved: kept so a save doesn't drop it.
        scrapeTimeout: '10s',
      },
    });
    expect(postprocessSectionForm(values, form)).toEqual({
      parameters: { ...saved.parameters, interval: '30s' },
    });
  });
});
