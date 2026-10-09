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

import { fireEvent, render, screen } from '@testing-library/react';
import { renderComponent } from './utils';
import { AffinityOperator } from 'shared-types/affinity.types';
import {
  Component,
  ComponentGroup,
  FieldType,
  GroupType,
  WIDGET_UI_TYPE,
  WidgetType,
} from 'components/ui-generator/ui-generator.types';
import { TOGGLEABLE_SWITCHES_KEY } from 'components/ui-generator/utils/toggleable/toggleable';
import { preprocessSchema } from 'components/ui-generator/utils/preprocess/preprocess-schema';
import { TestWrapper } from 'utils/test';

const makeSelectComponent = (path: string, label: string): Component => ({
  uiType: FieldType.Select,
  path,
  fieldParams: {
    label,
    options: [
      { label: 'Version 8.0', value: '8.0' },
      { label: 'Version 8.1', value: '8.1' },
    ],
  },
});

const makeTextComponent = (path: string, label: string): Component => ({
  uiType: FieldType.Text,
  path,
  fieldParams: { label },
});

describe('renderComponent - string values are shown correctly in preview', () => {
  it('renders a string value (e.g. databaseVersion) instead of "-"', () => {
    const component = makeSelectComponent(
      'spec.databaseVersion',
      'Database Version'
    );
    const formValues = { spec: { databaseVersion: '8.0' } };

    render(
      <TestWrapper>
        <>{renderComponent('databaseVersion', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Database Version: 8.0')).toBeInTheDocument();
  });

  it('shows "-" when the value is undefined', () => {
    const component = makeSelectComponent(
      'spec.databaseVersion',
      'Database Version'
    );
    const formValues = { spec: {} };

    render(
      <TestWrapper>
        <>{renderComponent('databaseVersion', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Database Version: -')).toBeInTheDocument();
  });

  it('shows "-" when the value is null', () => {
    const component = makeSelectComponent(
      'spec.databaseVersion',
      'Database Version'
    );
    const formValues = { spec: { databaseVersion: null } };

    render(
      <TestWrapper>
        <>{renderComponent('databaseVersion', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Database Version: -')).toBeInTheDocument();
  });

  it('renders a plain text value', () => {
    const component = makeTextComponent('spec.clusterName', 'Cluster Name');
    const formValues = { spec: { clusterName: 'my-cluster' } };

    render(
      <TestWrapper>
        <>{renderComponent('clusterName', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Cluster Name: my-cluster')).toBeInTheDocument();
  });

  it('renders boolean true as "Enabled"', () => {
    const component = makeTextComponent('spec.monitoring', 'Monitoring');
    const formValues = { spec: { monitoring: true } };

    render(
      <TestWrapper>
        <>{renderComponent('monitoring', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Monitoring: Enabled')).toBeInTheDocument();
  });

  it('renders boolean false as "Disabled"', () => {
    const component = makeTextComponent('spec.monitoring', 'Monitoring');
    const formValues = { spec: { monitoring: false } };

    render(
      <TestWrapper>
        <>{renderComponent('monitoring', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Monitoring: Disabled')).toBeInTheDocument();
  });

  it('renders a numeric value as a string', () => {
    const component = makeTextComponent('spec.replicas', 'Replicas');
    const formValues = { spec: { replicas: 3 } };

    render(
      <TestWrapper>
        <>{renderComponent('replicas', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Replicas: 3')).toBeInTheDocument();
  });

  it('appends the fieldParams badge to the value', () => {
    const component: Component = {
      uiType: FieldType.Number,
      path: 'spec.engine.storage.size',
      fieldParams: { label: 'Disk', badge: 'Gi' },
    };
    const formValues = { spec: { engine: { storage: { size: 25 } } } };

    render(
      <TestWrapper>
        <>{renderComponent('disk', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Disk: 25 Gi')).toBeInTheDocument();
  });

  it('does not append a badge when the value is missing', () => {
    const component: Component = {
      uiType: FieldType.Number,
      path: 'spec.engine.storage.size',
      fieldParams: { label: 'Disk', badge: 'Gi' },
    };
    const formValues = { spec: {} };

    render(
      <TestWrapper>
        <>{renderComponent('disk', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Disk: -')).toBeInTheDocument();
  });

  it('does not append a badge when the value is an empty string', () => {
    // coerceNumberInputValue returns '' for a cleared/untouched number
    // input, which is distinct from a missing (null/undefined) value.
    const component: Component = {
      uiType: FieldType.Number,
      path: 'spec.engine.storage.size',
      fieldParams: { label: 'Disk', badge: 'Gi' },
    };
    const formValues = { spec: { engine: { storage: { size: '' } } } };

    render(
      <TestWrapper>
        <>{renderComponent('disk', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Disk:')).toBeInTheDocument();
  });

  it('appends the badge for Select fields, matching the input', () => {
    const component = makeSelectComponent('spec.databaseVersion', 'Version');
    component.fieldParams = { ...component.fieldParams, badge: 'Gi' };
    const formValues = { spec: { databaseVersion: '8.0' } };

    render(
      <TestWrapper>
        <>{renderComponent('databaseVersion', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Version: 8.0 Gi')).toBeInTheDocument();
  });

  it('does not append a badge to an empty-string Select value', () => {
    const component = makeSelectComponent('spec.databaseVersion', 'Version');
    component.fieldParams = { ...component.fieldParams, badge: 'Gi' };
    const formValues = { spec: { databaseVersion: '' } };

    render(
      <TestWrapper>
        <>{renderComponent('databaseVersion', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Version:')).toBeInTheDocument();
  });

  it('does not append a badge for Text fields, matching the input', () => {
    const component = makeTextComponent('spec.clusterName', 'Cluster Name');
    component.fieldParams = { ...component.fieldParams, badge: 'Gi' };
    const formValues = { spec: { clusterName: 'my-cluster' } };

    render(
      <TestWrapper>
        <>{renderComponent('clusterName', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Cluster Name: my-cluster')).toBeInTheDocument();
  });

  it('does not append a badge for a Toggle field even if one is configured', () => {
    const component: Component = {
      uiType: FieldType.Toggle,
      path: 'spec.monitoring',
      fieldParams: { label: 'Monitoring', badge: 'Gi' },
    };
    const formValues = { spec: { monitoring: true } };

    render(
      <TestWrapper>
        <>{renderComponent('monitoring', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Monitoring: Enabled')).toBeInTheDocument();
  });

  it('does not append a badge to a non-numeric leftover string while a Number field is mid-edit', () => {
    // coerceNumberInputValue leaves the raw string in place when it fails to
    // parse as a number (e.g. the user has typed "1a" and not yet corrected it).
    const component: Component = {
      uiType: FieldType.Number,
      path: 'spec.engine.storage.size',
      fieldParams: { label: 'Disk', badge: 'Gi' },
    };
    const formValues = { spec: { engine: { storage: { size: '1a' } } } };

    render(
      <TestWrapper>
        <>{renderComponent('disk', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Disk: 1a')).toBeInTheDocument();
  });

  it('appends the badge when a Number field holds a valid numeric string', () => {
    // e.g. edit-mode default values extracted via extractInstanceValues, which
    // strips the badge suffix but doesn't coerce the remainder to a number.
    const component: Component = {
      uiType: FieldType.Number,
      path: 'spec.engine.storage.size',
      fieldParams: { label: 'Disk', badge: 'Gi' },
    };
    const formValues = { spec: { engine: { storage: { size: '25' } } } };

    render(
      <TestWrapper>
        <>{renderComponent('disk', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Disk: 25 Gi')).toBeInTheDocument();
  });

  it('does not append a badge for a NaN Number value', () => {
    const component: Component = {
      uiType: FieldType.Number,
      path: 'spec.engine.storage.size',
      fieldParams: { label: 'Disk', badge: 'Gi' },
    };
    const formValues = { spec: { engine: { storage: { size: NaN } } } };

    render(
      <TestWrapper>
        <>{renderComponent('disk', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Disk: NaN')).toBeInTheDocument();
  });

  it('does not append a badge for an Infinity Number value', () => {
    const component: Component = {
      uiType: FieldType.Number,
      path: 'spec.engine.storage.size',
      fieldParams: { label: 'Disk', badge: 'Gi' },
    };
    const formValues = {
      spec: { engine: { storage: { size: Infinity } } },
    };

    render(
      <TestWrapper>
        <>{renderComponent('disk', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Disk: Infinity')).toBeInTheDocument();
  });

  it('uses first path from multipath field for preview lookup', () => {
    const component: Component = {
      uiType: FieldType.Text,
      path: ['spec.engine.version', 'spec.proxy.version'],
      fieldParams: { label: 'Version' },
    };
    const formValues = {
      spec: {
        engine: { version: '8.0.41' },
      },
    };

    render(
      <TestWrapper>
        <>{renderComponent('engineVersion', component, formValues)}</>
      </TestWrapper>
    );

    expect(screen.getByText('Version: 8.0.41')).toBeInTheDocument();
  });
});

describe('renderComponent - toggleable group', () => {
  const rawMonitoring: ComponentGroup = {
    uiType: 'group',
    groupType: GroupType.Toggleable,
    label: 'Monitoring',
    components: {
      endpoint: makeTextComponent('spec.monitoring.endpoint', 'Endpoint'),
    },
  };
  const monitoring = preprocessSchema({
    replicaSet: {
      sections: { advanced: { components: { monitoring: rawMonitoring } } },
    },
  }).replicaSet.sections.advanced.components.monitoring;

  const renderWithSwitch = (switchOn: boolean) =>
    render(
      <TestWrapper>
        <>
          {renderComponent('monitoring', monitoring, {
            [TOGGLEABLE_SWITCHES_KEY]: {
              'advanced~monitoring': switchOn,
            },
            spec: { monitoring: { endpoint: 'pmm:443' } },
          })}
        </>
      </TestWrapper>
    );

  it('summarises a switched-off section as disabled, without its fields', () => {
    renderWithSwitch(false);

    expect(screen.getByText('Monitoring: Disabled')).toBeInTheDocument();
    expect(screen.queryByText(/Endpoint/)).not.toBeInTheDocument();
  });

  it('lists the fields of a switched-on section', () => {
    renderWithSwitch(true);

    expect(screen.getByText('Endpoint: pmm:443')).toBeInTheDocument();
    expect(screen.queryByText('Monitoring: Disabled')).not.toBeInTheDocument();
  });
});

describe('renderComponent - widget markers', () => {
  const enginePath = 'spec.components.engine.schedulingPolicy.affinity';
  const marker: Component = {
    uiType: WIDGET_UI_TYPE,
    widgetType: WidgetType.PodSchedulingPolicy,
    id: 'podSchedulingPolicy',
    _widgetTargets: [
      { key: 'engine', path: enginePath },
      { key: 'proxy', path: 'spec.components.proxy.schedulingPolicy.affinity' },
    ],
  };
  const engineAffinity = {
    nodeAffinity: {
      requiredDuringSchedulingIgnoredDuringExecution: {
        nodeSelectorTerms: [
          {
            matchExpressions: [
              {
                key: 'disktype',
                operator: AffinityOperator.In,
                values: ['ssd'],
              },
            ],
          },
        ],
      },
    },
  };

  const renderMarker = (formValues: Record<string, unknown>) =>
    render(
      <TestWrapper>
        <>{renderComponent('podSchedulingPolicy', marker, formValues)}</>
      </TestWrapper>
    );

  it('digests rules per component and opens the full view on demand', () => {
    renderMarker({
      spec: {
        components: {
          engine: { schedulingPolicy: { affinity: engineAffinity } },
        },
      },
    });

    expect(screen.getByText('engine: 1 required')).toBeInTheDocument();
    expect(screen.queryByText(/^proxy:/)).not.toBeInTheDocument();
    expect(screen.queryByText('disktype')).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId('preview-widget-toggle'));

    expect(screen.getByText('disktype')).toBeInTheDocument();
  });

  it('shows a dash when no component has rules', () => {
    renderMarker({ spec: {} });

    expect(screen.getByText('Pod scheduling policy: -')).toBeInTheDocument();
    expect(
      screen.queryByTestId('preview-widget-toggle')
    ).not.toBeInTheDocument();
  });
});
