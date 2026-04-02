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
import type { ConnectionSectionProps } from './connection-section.types';

const ConnectionSection = ({
  credentials,
  loading,
}: ConnectionSectionProps) => (
  <OverviewSection
    dataTestId="connection-details"
    title={Messages.titles.connectionDetails}
    loading={loading}
  >
    {credentials ? (
      <>
        <OverviewSectionRow
          label={Messages.fields.host}
          content={credentials.host}
        />
        <OverviewSectionRow
          label={Messages.fields.port}
          content={credentials.port}
        />
        <OverviewSectionRow
          label={Messages.fields.username}
          content={credentials.username}
        />
        <OverviewSectionRow
          label={Messages.fields.connectionUrl}
          content={credentials.uri}
        />
      </>
    ) : (
      <OverviewSectionRow
        label={Messages.fields.status}
        content="Waiting for instance to be ready..."
      />
    )}
  </OverviewSection>
);

export default ConnectionSection;
