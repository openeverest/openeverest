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
  FieldType,
  FormMode,
  GroupType,
  WIDGET_UI_TYPE,
  WidgetType,
} from 'components/ui-generator/ui-generator.types';
import { TOGGLEABLE_SWITCHES_KEY } from '../toggleable/toggleable';
import { deepClone, setByPath } from '../object-path';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { extractInstanceValues } from '../default-values/extract-instance-values';
import { mergeSectionEdit } from './merge-section-edit';

const { sections } = preprocessSchema({
  replicaSet: {
    sections: {
      advanced: {
        components: {
          monitoring: {
            uiType: 'group',
            groupType: GroupType.Toggleable,
            components: {
              url: {
                uiType: FieldType.Text,
                path: 'spec.monitoring.url',
                fieldParams: { label: 'URL' },
              },
            },
          },
          replicas: {
            uiType: FieldType.Number,
            path: 'spec.replicas',
            fieldParams: { label: 'Replicas' },
          },
        },
      },
    },
  },
}).replicaSet;

const savedSpec = () => ({
  replicas: 3,
  monitoring: { url: 'http://pmm' },
  backup: { enabled: true },
});

const merge = (switchOn: boolean, spec = savedSpec()) =>
  mergeSectionEdit({
    spec,
    sections,
    sectionKey: 'advanced',
    topology: 'replicaSet',
    initialValues: {
      [TOGGLEABLE_SWITCHES_KEY]: { 'advanced~monitoring': true },
      spec: { replicas: 3, monitoring: { url: 'http://pmm' } },
    },
    formData: {
      [TOGGLEABLE_SWITCHES_KEY]: { 'advanced~monitoring': switchOn },
      spec: { replicas: 5, monitoring: { url: 'http://pmm-2' } },
    },
  });

describe('mergeSectionEdit', () => {
  it('applies the section values and keeps fields it does not own', () => {
    expect(merge(true)).toEqual({
      replicas: 5,
      monitoring: { url: 'http://pmm-2' },
      backup: { enabled: true },
    });
  });

  it('does not mutate the saved spec', () => {
    const spec = savedSpec();
    merge(false, spec);
    expect(spec).toEqual(savedSpec());
  });

  describe('saved values the form reads in another form', () => {
    const quantitySections = preprocessSchema({
      t: {
        sections: {
          resources: {
            components: {
              cpu: {
                uiType: FieldType.Number,
                path: 'spec.cpu',
                fieldParams: { label: 'CPU' },
              },
              replicas: {
                uiType: FieldType.Number,
                path: 'spec.replicas',
                fieldParams: { label: 'Replicas' },
              },
            },
          },
        },
      },
    }).t.sections;
    const saved = { cpu: '600m', replicas: 3 };

    const save = (changes: Record<string, unknown>) => {
      const initialValues = extractInstanceValues(
        quantitySections,
        { spec: saved },
        FormMode.Edit
      );
      const formData = deepClone(initialValues);
      Object.entries(changes).forEach(([key, value]) =>
        setByPath(formData, `spec.${key}`, value)
      );
      return mergeSectionEdit({
        spec: saved,
        sections: quantitySections,
        sectionKey: 'resources',
        topology: 't',
        initialValues,
        formData,
      });
    };

    it('keeps an untouched quantity as saved, not as the number it reads as', () => {
      expect(save({ replicas: 5 })).toEqual({ cpu: '600m', replicas: 5 });
    });

    it('writes a quantity the user changed', () => {
      expect(save({ cpu: 2 })).toEqual({ cpu: 2, replicas: 3 });
    });
  });

  describe('widget values', () => {
    const widgetSections = {
      scheduling: {
        components: {
          affinity: {
            uiType: WIDGET_UI_TYPE,
            widgetType: WidgetType.Affinity,
            path: 'spec.affinity',
          },
        },
      },
    };
    const savedAffinity = {
      replicas: 3,
      affinity: { nodeAffinity: { a: 1 }, podAffinity: { b: 2 } },
    };
    const mergeWidget = (affinity: unknown) =>
      mergeSectionEdit({
        spec: savedAffinity,
        sections: widgetSections,
        sectionKey: 'scheduling',
        initialValues: { spec: { affinity: savedAffinity.affinity } },
        formData: { spec: { affinity } },
      });

    it('replaces the saved value instead of merging into it', () => {
      expect(mergeWidget({ podAffinity: { b: 3 } })).toEqual({
        replicas: 3,
        affinity: { podAffinity: { b: 3 } },
      });
    });

    it('deletes the saved value when the widget is cleared', () => {
      expect(mergeWidget({})).toEqual({ replicas: 3 });
    });
  });
});
