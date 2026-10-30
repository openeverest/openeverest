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

import { PreviewContentText } from '../preview-section';
import { SectionProps } from './section.types';
import { parseMonitoringDestinations } from '../../database-form-body/steps/monitoring-step';
import { Messages } from '../../database-form-body/steps/monitoring-step/monitoring-step.messages';

export const PreviewMonitoringSection = (props: SectionProps) => {
  const monitoring: unknown = Reflect.get(props, 'monitoring');
  const destinations = parseMonitoringDestinations(
    typeof monitoring === 'object' && monitoring !== null
      ? Reflect.get(monitoring, 'destinations')
      : undefined
  );

  if (destinations.length === 0) {
    return <PreviewContentText text={Messages.previewNone} />;
  }

  return (
    <>
      {destinations.map((entry) => (
        <PreviewContentText key={entry.name} text={entry.destinationRef.name} />
      ))}
    </>
  );
};
