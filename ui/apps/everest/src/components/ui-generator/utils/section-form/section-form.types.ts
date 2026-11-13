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

import type { Section, TopologyUISchemas } from '../../ui-generator.types';

// One section rendered as a form of its own (e.g. a class's form), prepared by
// the same pipeline stages the ui-generator runs on a provider topology.
export interface SectionForm {
  sectionKey: string;
  schema: TopologyUISchemas;
  sections: Record<string, Section>;
}
