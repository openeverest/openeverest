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

const isUpperCase = (char = '') => char >= 'A' && char <= 'Z';
const isLowerCase = (char = '') => char >= 'a' && char <= 'z';

// A capital starts a word unless it continues an acronym: myURLParser → my-url-parser.
const hyphenateCapitals = (str: string) =>
  str
    .split('')
    .map((char, index) => {
      if (!isUpperCase(char)) {
        return char;
      }
      const startsWord =
        index > 0 &&
        (!isUpperCase(str[index - 1]) || isLowerCase(str[index + 1]));
      return (startsWord ? '-' : '') + char.toLowerCase();
    })
    .join('');

// A whitespace run becomes one hyphen, or disappears next to an existing one.
const hyphenateWhitespace = (str: string) =>
  str.replace(/\s+/g, (run: string, offset: number) =>
    str[offset - 1] === '-' || str[offset + run.length] === '-' ? '' : '-'
  );

const kebabize = (str: string) =>
  hyphenateWhitespace(hyphenateCapitals(str.trim()));

export const pluralize = (
  count: number,
  singular: string,
  plural: string = `${singular}s`
): string => `${count} ${count === 1 ? singular : plural}`;

export default kebabize;
