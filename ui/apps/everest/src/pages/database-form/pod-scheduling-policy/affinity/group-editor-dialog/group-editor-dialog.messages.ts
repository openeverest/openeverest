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

export const Messages = {
  addGroup: 'Add rule group',
  editGroup: 'Edit rule group',
  addGroupTo: (component: string) =>
    `Add rule group to the ${component} component`,
  editGroupFor: (component: string) =>
    `Edit rule group for the ${component} component`,
  add: 'Add',
  save: 'Save',
  description:
    'Create a group of affinity rules to control how your database workloads are allocated across your system. Use it to enhance performance, improve resource management, or ensure high availability based on your deployment needs.',
  ruleType: 'Rule type',
  typeHelper: 'How pods are scheduled',
  conditions: 'Conditions',
  conditionsInfo:
    'Conditions inside a group are combined with AND: a match needs all of them.',
  valuesHelper: 'Separate multiple values with commas',
  keyPlaceholder: 'Key',
  valuesLabel: 'Values',
  valuesPlaceholder: 'e.g. ssd, nvme',
  invalidLabelValues:
    "Values may use letters, numbers, '-', '_', '.' (max 63 chars each)",
  conditionsRequired: 'At least one condition is required',
  topologyKeyLabel: 'Topology Key',
  numericValuePlaceholder: 'e.g. 4',
  numericValueInvalid: 'Enter a single whole number',
  numericOperatorNodeOnly:
    'Greater than / less than work only with node affinity',
  addCondition: 'Add condition',
  removeCondition: 'Remove condition',
};
