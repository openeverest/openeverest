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
  Section,
  WidgetTarget,
  WIDGET_UI_TYPE,
  WidgetType,
} from 'components/ui-generator/ui-generator.types';

// One section per resolved target, each holding a single affinity widget bound
// to that component's schedulingPolicy.affinity path.
export const buildAffinitySections = (
  targets: WidgetTarget[]
): Record<string, Section> =>
  Object.fromEntries(
    targets.map(({ key, path }) => [
      key,
      {
        components: {
          affinity: {
            uiType: WIDGET_UI_TYPE,
            widgetType: WidgetType.Affinity,
            path,
          },
        },
      },
    ])
  );
