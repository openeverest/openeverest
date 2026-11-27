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

// A quoted string is not a path: a map key like 'example.com/role' would
// otherwise be watched as a form field.
const STRING_LITERAL = /'(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*"/g;

export const extractCelFieldPaths = (celExpr: string): string[][] => {
  // Regex matches patterns like: word.word.word (at least 2 segments)
  const regex = /([a-zA-Z0-9_]+(?:\.[a-zA-Z0-9_]+)+)/g;
  const matches = celExpr.replace(STRING_LITERAL, "''").match(regex) || [];

  // Remove duplicates and split each path into an array
  return Array.from(new Set(matches)).map((field) => field.split('.'));
};
