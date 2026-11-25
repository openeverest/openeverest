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
import { act, fireEvent, render, screen } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import {
  FieldType,
  type KeyValueWidget,
  TemplateWidgetType,
  TopologyUISchemas,
  WIDGET_UI_TYPE,
} from '../ui-generator.types';
import { getDefaultValues } from '../utils/default-values';

const INPUT_WIDTH = 100;
const LONG_KEY = 'service.beta.kubernetes.io/aws-load-balancer-internal';

const schemaWith = (
  fieldParams: KeyValueWidget['fieldParams'] = {}
): TopologyUISchemas => ({
  replicaSet: {
    sections: {
      advanced: {
        components: {
          annotations: {
            uiType: FieldType.Map,
            path: 'spec.annotations',
            fieldParams: {
              label: 'Annotations',
              defaultValue: { [LONG_KEY]: 'true' },
            },
            itemTemplate: {
              uiType: WIDGET_UI_TYPE,
              widgetType: TemplateWidgetType.KeyValue,
              fieldParams,
            },
          },
        },
      },
    },
  },
});

const FormWrapper = ({
  children,
  uiSchema,
}: {
  children: ReactNode;
  uiSchema: TopologyUISchemas;
}) => {
  const methods = useForm({
    defaultValues: getDefaultValues(uiSchema, 'replicaSet'),
  });
  return <FormProvider {...methods}>{children}</FormProvider>;
};

const renderMap = (uiSchema = schemaWith()) =>
  render(
    <TestWrapper>
      <FormWrapper uiSchema={uiSchema}>
        <UIGenerator
          sections={uiSchema.replicaSet.sections}
          sectionKey="advanced"
        />
      </FormWrapper>
    </TestWrapper>
  );

const flushTimers = () =>
  act(() => {
    vi.runAllTimers();
  });

const hover = (input: HTMLElement) => {
  fireEvent.mouseOver(input);
  flushTimers();
};

const leave = (input: HTMLElement) => {
  fireEvent.mouseLeave(input);
  flushTimers();
};

const shownTooltips = () =>
  screen.queryAllByRole('tooltip').map((tooltip) => tooltip.textContent);

describe('UIGenerator - a map value its input cuts off', () => {
  // jsdom lays nothing out, so every element reports the same width and only
  // an input reports content wider than that.
  beforeAll(() => {
    vi.spyOn(Element.prototype, 'clientWidth', 'get').mockReturnValue(
      INPUT_WIDTH
    );
    vi.spyOn(Element.prototype, 'scrollWidth', 'get').mockImplementation(
      function (this: Element) {
        return this.tagName === 'INPUT' ? INPUT_WIDTH * 3 : INPUT_WIDTH;
      }
    );
  });

  afterAll(() => {
    vi.restoreAllMocks();
  });

  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows the value on hover, as it stands in the input', () => {
    renderMap();
    const key = screen.getByRole('textbox', { name: 'Key 1' });

    fireEvent.change(key, { target: { value: `${LONG_KEY}-edited` } });
    fireEvent.blur(key);
    hover(key);

    expect(shownTooltips()).toEqual([`${LONG_KEY}-edited`]);
  });

  it('stays out of the way while the value is edited', () => {
    renderMap();
    const key = screen.getByRole('textbox', { name: 'Key 1' });

    fireEvent.focus(key);
    hover(key);

    expect(shownTooltips()).toEqual([]);
  });

  it('does not come back once the edit ends with the pointer elsewhere', () => {
    renderMap();
    const key = screen.getByRole('textbox', { name: 'Key 1' });

    hover(key);
    fireEvent.focus(key);
    leave(key);
    fireEvent.blur(key);
    flushTimers();

    expect(shownTooltips()).toEqual([]);

    hover(key);

    expect(shownTooltips()).toEqual([LONG_KEY]);
  });

  it('never shows a masked value', () => {
    renderMap(schemaWith({ value: { type: 'password' } }));

    hover(screen.getByLabelText('Value 1'));

    expect(shownTooltips()).toEqual([]);
  });

  it('leaves the hover to a tooltip the schema sets', () => {
    renderMap(schemaWith({ key: { tooltip: 'A DNS prefix, then a name' } }));

    hover(screen.getByRole('textbox', { name: 'Key 1' }));

    expect(shownTooltips()).toEqual(['A DNS prefix, then a name']);
  });
});
