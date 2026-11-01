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

const kebabize = (str: string) =>
  str
    .replace(
      /[A-Z]+(?![a-z])|[A-Z]/g,
      ($, ofs) => (ofs ? '-' : '') + $.toLowerCase()
    )
    .replace(/\s+(?=[-])/, '')
    .replace(/\s+(?![-])/, '-');

export const pluralize = (
  count: number,
  singular: string,
  plural: string = `${singular}s`
): string => `${count} ${count === 1 ? singular : plural}`;

export default kebabize;
