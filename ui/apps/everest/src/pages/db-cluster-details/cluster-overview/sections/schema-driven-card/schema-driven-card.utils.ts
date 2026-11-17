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

import type { SectionField } from '../../utils/cluster-overview.helpers';

export type CardSegment =
  | { kind: 'rows'; fields: SectionField[] }
  | { kind: 'summary'; field: SectionField };

// Consecutive scalar fields share one block; each widget summary gets its own
// headed block, keeping the schema's field order.
export const segmentCardFields = (fields: SectionField[]): CardSegment[] =>
  fields.reduce<CardSegment[]>((segments, field) => {
    const last = segments[segments.length - 1];
    if (field.summary) {
      segments.push({ kind: 'summary', field });
    } else if (last?.kind === 'rows') {
      last.fields.push(field);
    } else {
      segments.push({ kind: 'rows', fields: [field] });
    }
    return segments;
  }, []);
