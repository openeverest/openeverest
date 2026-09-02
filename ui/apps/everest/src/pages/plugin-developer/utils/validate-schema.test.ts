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

import { describe, it, expect } from 'vitest';
import { validateSchema } from './validate-schema';

describe('validateSchema', () => {
  it('returns no diagnostics and a parsed object for a valid schema', () => {
    const yaml = [
      'standalone:',
      '  sections:',
      '    basics:',
      '      components:',
      '        nodes:',
      '          uiType: number',
      '          path: x',
      '',
    ].join('\n');
    const r = validateSchema(yaml);
    expect(r.diagnostics).toEqual([]);
    expect(r.parsed).not.toBeNull();
  });

  it('reports a YAML syntax error with a line and null parsed', () => {
    // Tab indentation is invalid in YAML.
    const yaml = 'standalone:\n\t- bad';
    const r = validateSchema(yaml);
    expect(r.parsed).toBeNull();
    expect(r.diagnostics.length).toBeGreaterThan(0);
    expect(r.diagnostics[0].severity).toBe('error');
    expect(r.diagnostics[0].line).toBeGreaterThanOrEqual(1);
  });

  it('points a wrong-value error at the offending node', () => {
    const lines = [
      'standalone:',
      '  sections:',
      '    basics:',
      '      components:',
      '        nodes:', // line 5 — the component key
      '          uiType: integer', // line 6 — invalid value
      '',
    ];
    const r = validateSchema(lines.join('\n'));
    expect(r.parsed).toBeNull();
    expect(r.diagnostics).toHaveLength(1);
    // With Fix 1, an invalid uiType makes the whole component union fail; the
    // union issue's path stops at "nodes" (a Map), so the squiggle lands on the
    // component KEY at line 5 rather than the value at line 6.
    expect(r.diagnostics[0].line).toBe(5);
    expect(r.diagnostics[0].message).toMatch(/uiType/);
  });

  it('falls back to the parent node for a missing required key', () => {
    const lines = [
      'standalone:',
      '  sections:',
      '    basics:',
      '      components:',
      '        nodes:', // line 5 — the parent component, has no uiType
      '          path: x',
      '',
    ];
    const r = validateSchema(lines.join('\n'));
    expect(r.parsed).toBeNull();
    expect(r.diagnostics).toHaveLength(1);
    // The missing "uiType" key has no node of its own, so the Zod union issue
    // resolves to the nearest existing ancestor: the "nodes" component. With
    // the installed Zod, the union issue's `path` stops at "nodes", so
    // `nearestExistingNode` returns the "nodes" map value node, whose range
    // starts at its first child ("path: x") on line 6 — still the existing
    // "nodes" ancestor. (See validate-schema spike note.)
    expect(r.diagnostics[0].line).toBe(5);
    expect(r.diagnostics[0].message).toMatch(/uiType|component/i);
  });

  it('flags a group missing its components map on the group key', () => {
    const lines = [
      'standalone:',
      '  sections:',
      '    basics:',
      '      components:',
      '        grp:',
      '          uiType: group',
      '',
    ];
    const r = validateSchema(lines.join('\n'));
    expect(r.parsed).toBeNull();
    expect(r.diagnostics.length).toBeGreaterThanOrEqual(1);
    expect(r.diagnostics[0].line).toBe(5); // the `grp:` key, not its value
  });

  it('reports a diagnostic per invalid component', () => {
    const lines = [
      'standalone:',
      '  sections:',
      '    basics:',
      '      components:',
      '        a:',
      '          uiType: integer',
      '        b:',
      '          uiType: bogus',
      '',
    ];
    const r = validateSchema(lines.join('\n'));
    expect(r.parsed).toBeNull();
    expect(r.diagnostics.length).toBeGreaterThanOrEqual(2);
  });

  it('returns a top-level diagnostic for empty input', () => {
    const r = validateSchema('');
    expect(r.parsed).toBeNull();
    expect(r.diagnostics).toHaveLength(1);
    expect(r.diagnostics[0].message).toMatch(/object/i);
  });

  it('returns a top-level diagnostic for a non-object root', () => {
    const r = validateSchema('just a string');
    expect(r.parsed).toBeNull();
    expect(r.diagnostics[0].message).toMatch(/object/i);
  });
});
