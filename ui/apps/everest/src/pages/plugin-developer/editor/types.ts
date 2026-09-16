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

export type EditorTheme = 'light' | 'dark';

export type DiagnosticSeverity = 'error' | 'warning';

// A validation finding with 1-based line/column positions.
export type Diagnostic = {
  line: number;
  column: number;
  endLine?: number;
  endColumn?: number;
  message: string;
  severity: DiagnosticSeverity;
};

// Convert a 1-based line/column into a 0-based character offset in `text`,
// clamping anything out of range into the document. Used by CodeMirrorEditor to
// place lint ranges, and unit-tested here in isolation.
export const lineColToOffset = (
  text: string,
  line: number,
  column: number
): number => {
  const lines = text.split('\n');
  const targetLine = Math.min(Math.max(line, 1), lines.length);
  let offset = 0;
  for (let i = 0; i < targetLine - 1; i++) {
    offset += lines[i].length + 1; // +1 for the consumed '\n'
  }
  const lineLength = lines[targetLine - 1].length;
  const col = Math.min(Math.max(column - 1, 0), lineLength);
  return Math.min(offset + col, text.length);
};
