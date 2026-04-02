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

import OverviewSection from '../../overview-section';
import OverviewSectionRow from '../../overview-section-row';
import { Messages } from '../../cluster-overview.messages';
import type { BasicInfoSectionProps } from './basic-info-section.types';

const BasicInfoSection = ({
  instance,
  namespace,
  loading,
}: BasicInfoSectionProps) => (
  <OverviewSection
    dataTestId="basic-information"
    title={Messages.titles.basicInformation}
    loading={loading}
  >
    <OverviewSectionRow
      label={Messages.fields.name}
      content={instance.metadata?.name}
    />
    <OverviewSectionRow label={Messages.fields.namespace} content={namespace} />
    <OverviewSectionRow label="Provider" content={instance.spec?.provider} />
    <OverviewSectionRow
      label="Topology"
      content={instance.spec?.topology?.type ?? 'default'}
    />
    <OverviewSectionRow
      label={Messages.fields.status}
      content={instance.status?.phase}
    />
  </OverviewSection>
);

export default BasicInfoSection;
