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

export type ResourceRequestsSectionProps = {
  switchName: string;
  synced: boolean;
  cpuInputName: string;
  memoryInputName: string;
  cpuLimitName: string;
  memoryLimitName: string;
  numberOfUnitsName: string;
  customNrOfUnitsName: string;
  unit: string;
  unitPlural: string;
  // Whether the limits above include a DISK column. When true, an empty third
  // slot keeps the CPU/Memory request fields aligned with the limits; when
  // false (e.g. proxies without disk) the fields fill the row evenly instead.
  hasDiskColumn?: boolean;
};
