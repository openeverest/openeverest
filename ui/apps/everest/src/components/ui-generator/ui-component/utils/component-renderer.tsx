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

import React from 'react';
import { Component, FieldType } from '../../ui-generator.types';
import { renderSelectOptions } from './select-component-handler';
import { Provider } from 'shared-types/api.types';

export const renderComponentChildren = (
  item: Component,
  name: string,
  providerObject?: Provider
): React.ReactNode | undefined => {
  switch (item.uiType) {
    case FieldType.Select:
      return renderSelectOptions(item, name, providerObject);

    case FieldType.Number:
    case FieldType.Hidden:
      return undefined;

    default:
      return undefined;
  }
};
