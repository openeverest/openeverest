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

import { type Meta, type StoryObj } from '@storybook/react';
import CodeCopyBlock from './code-copy-block';
import { CodeCopyBlockProps } from './code-copy-block.types';

const meta = {
  title: 'CodeCopyBlock',
  component: CodeCopyBlock,
  parameters: {
    layout: 'centered',
  },
} satisfies Meta<CodeCopyBlockProps>;

export default meta;
type Story = StoryObj<CodeCopyBlockProps>;

export const WithCopyButtonCommand: Story = {
  args: {
    message:
      'helm install everest openeverest/everest-db-namespace --create-namespace --namespace <NAMESPACE>',
    showCopyButtonText: true,
  },
};
export const WithoutCopyButtonCommand: Story = {
  args: {
    message:
      'helm install everest openeverest/everest-db-namespace --create-namespace --namespace <NAMESPACE>',
    showCopyButtonText: false,
  },
};
export const Warning: Story = {
  args: {
    message:
      "configServer: 2 of 3 pods cannot be scheduled: 0/4 nodes are available: 3 node(s) didn't match pod anti-affinity rules.\nengine: 1 of 3 pods cannot be scheduled: 0/4 nodes are available: 3 node(s) didn't match pod anti-affinity rules.",
    severity: 'warning',
  },
};
