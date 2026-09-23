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

import { BackupsPreviewSection } from './backups-section.js';
import { AdvancedConfigurationsPreviewSection } from './advanced-configurations-section.js';
import { PreviewSectionOne } from '../../database-preview/sections/base-step.js';
import { useLocation } from 'react-router-dom';
import { PreviewContentText } from '../../database-preview/preview-section.js';

export const usePreviewSections = () => {
  const location = useLocation();
  const showImportStep = location.state?.showImport;
  return [
    { component: PreviewSectionOne, title: 'Basic Information' },
    ...(showImportStep
      ? [
          {
            component: () => PreviewContentText({ text: '' }),
            title: 'Import information',
          },
        ]
      : []),
    { component: BackupsPreviewSection, title: 'Backups' },
    {
      component: AdvancedConfigurationsPreviewSection,
      title: 'Advanced Configurations',
    },
  ];
};
