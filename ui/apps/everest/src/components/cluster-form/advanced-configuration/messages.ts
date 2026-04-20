// everest
// Copyright (C) 2023 Percona LLC
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
  DUPLICATE_SOURCE_RANGE_ERROR,
  INVALID_SOURCE_RANGE_ERROR,
  SOURCE_RANGE_PLACEHOLDER,
} from 'consts';

export const Messages = {
  enable: 'Enable',
  sourceRangePlaceholder: SOURCE_RANGE_PLACEHOLDER,
  cards: {
    storage: {
      title: 'Storage',
      description:
        'Defines the type and performance of storage for your database. Select based on workload needs, such as high IOPS for fast access or cost-effective options for less frequent use.',
    },
    policies: {
      title: 'Pod scheduling policy',
      description: 'Select one of the available pod scheduling policies.',
    },
    enableExternalAccess: {
      title: 'External Access',
      description: `
        Select the desired Exposure Method and provide the appropriate configuration to enable access.
      `,
    },
    exposureMethod: {
      title: 'Exposure Method',
    },
    loadBalancerConfiguration: {
      title: 'LoadBalancer configuration',
    },
    sourceRange: {
      title: 'Source Range',
      description:
        'Specify trusted IP addresses to restrict access. Leaving this blank will expose the database to all IP addresses.',
    },
    splitHorizonDNS: {
      title: 'Split-Horizon DNS',
      description:
        'Choose a domain name for your cluster. Everest will auto-generate replica hostnames for each network horizon.',
    },
  },
  errors: {
    sourceRange: {
      invalid: INVALID_SOURCE_RANGE_ERROR,
      duplicate: DUPLICATE_SOURCE_RANGE_ERROR,
    },
  },
  tooltipTexts: {
    noPolicies:
      'Seems like you don’t have permission to read any pod scheduling policy.',
    limitations: 'Check limitations in the documentation',
    splitHorizonDNS: (domain: string) =>
      `OpenEverest will create domains using ${domain} as suffix`,
  },
};
