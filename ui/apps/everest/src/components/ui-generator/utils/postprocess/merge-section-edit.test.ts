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
  GroupType,
  WIDGET_UI_TYPE,
  WidgetType,
} from 'components/ui-generator/ui-generator.types';
import { TOGGLEABLE_SWITCHES_KEY } from '../toggleable/toggleable';
import { preprocessSchema } from '../preprocess/preprocess-schema';
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

  it('deletes a switched-off group, leaving no empty parent behind', () => {
    expect(merge(false)).toEqual({ replicas: 5, backup: { enabled: true } });
  });

  it('does not mutate the saved spec', () => {
    const spec = savedSpec();
    merge(false, spec);
    expect(spec).toEqual(savedSpec());
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
