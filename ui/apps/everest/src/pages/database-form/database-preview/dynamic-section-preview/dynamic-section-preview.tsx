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
import { renderComponent } from './utils';
import { orderComponents } from 'components/ui-generator/utils/component-renderer';
import { Section } from 'components/ui-generator/ui-generator.types';

export const DynamicSectionPreview = ({
  section,
  formValues,
}: {
  section: Section;
  formValues: Record<string, unknown>;
}) => {
  const sectionComponents = section?.components;
  if (!sectionComponents || typeof sectionComponents !== 'object') {
    return <PreviewContentText text="No data" />;
  }

  return (
    <>
      {orderComponents(sectionComponents, section?.componentsOrder).map(
        ([key, comp]) =>
          renderComponent(
            key,
            comp,
            formValues,
            `${section?.label || ''}.${key}`
          )
      )}
    </>
  );
};

export default DynamicSectionPreview;
