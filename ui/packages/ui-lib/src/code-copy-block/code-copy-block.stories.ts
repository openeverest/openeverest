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
