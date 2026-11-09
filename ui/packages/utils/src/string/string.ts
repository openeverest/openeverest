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

// Checks the hyphen after a run here, not in a lookahead, to stay linear.
const replaceFirstWhitespace = (
  str: string,
  replacer: (run: string, beforeHyphen: boolean) => string
) => {
  let replaced = false;
  return str.replace(/\s+/g, (run: string, offset: number) => {
    if (replaced) {
      return run;
    }
    const result = replacer(run, str[offset + run.length] === '-');
    replaced = result !== run;
    return result;
  });
};

// Only the first whitespace run is joined: e2e data-testids rely on it.
const kebabize = (str: string) =>
  replaceFirstWhitespace(
    replaceFirstWhitespace(hyphenateCapitals(str), (run, beforeHyphen) =>
      beforeHyphen ? '' : run
    ),
    (run, beforeHyphen) => {
      if (!beforeHyphen) {
        return '-';
      }
      return run.length > 1 ? `-${run.slice(-1)}` : run;
    }
  );

export const pluralize = (
  count: number,
  singular: string,
  plural: string = `${singular}s`
): string => `${count} ${count === 1 ? singular : plural}`;

export default kebabize;
