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

import { render, screen } from '@testing-library/react';
import { renderComponent } from './utils';
import {
  Component,
  FieldType,
} from 'components/ui-generator/ui-generator.types';
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
