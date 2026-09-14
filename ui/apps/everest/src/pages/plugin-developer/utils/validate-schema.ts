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

import { parseDocument, LineCounter, isMap, isSeq, isScalar, Node } from 'yaml';
import { ZodIssue } from 'zod';
import { TopologyUISchemas } from 'components/ui-generator/ui-generator.types';
import {
  topologyUISchemasSchema,
  FIELD_UI_TYPES,
} from './topology-ui-schema.zod';
import { Diagnostic } from '../editor/types';

type ValidateResult = {
  diagnostics: Diagnostic[];
  parsed: TopologyUISchemas | null;
};

// Walk `node` following `path` and return the deepest node that actually
// exists. For a missing-key path, this is the nearest existing ancestor — the
// node we attach the squiggle to. Returns the key node when the final segment
// resolves to a map pair, so the squiggle lands on the key, not the value.
const nearestExistingNode = (
  root: unknown,
  path: (string | number)[]
): Node | null => {
  let current: unknown = root;
  let lastKeyNode: Node | null = null;

  for (const segment of path) {
    if (!isMap(current)) break;
    const pair = current.items.find(
      (p) => isScalar(p.key) && String(p.key.value) === String(segment)
    );
    if (!pair) break; // segment is missing -> stop at the last existing node
    lastKeyNode = (pair.key as Node) ?? null;
    current = pair.value;
  }
  // Underline the component KEY when the node is a collection (the error is
  // about a missing/invalid child), or the value itself when it's a scalar.
  if (
    current &&
    typeof current === 'object' &&
    'range' in (current as object)
  ) {
    if (isMap(current) || isSeq(current)) {
      return lastKeyNode ?? (current as Node);
    }
    return current as Node;
  }
  return lastKeyNode;
};

const offsetToLineCol = (
  counter: LineCounter,
  offset: number
): { line: number; column: number } => {
  const pos = counter.linePos(offset);
  return { line: pos.line, column: pos.col };
};

export const validateSchema = (yamlText: string): ValidateResult => {
  const counter = new LineCounter();
  const doc = parseDocument(yamlText, { lineCounter: counter });

  // Layer 1: YAML syntax errors.
  if (doc.errors.length > 0) {
    const diagnostics: Diagnostic[] = doc.errors.map((err) => {
      const start = err.pos?.[0] ?? 0;
      const { line, column } = offsetToLineCol(counter, start);
      return {
        line,
        column,
        message: err.message,
        severity: 'error' as const,
      };
    });
    return { diagnostics, parsed: null };
  }

  const json = doc.toJS();
  if (!json || typeof json !== 'object' || Array.isArray(json)) {
    return {
      diagnostics: [
        {
          line: 1,
          column: 1,
          message: 'Top level YAML value must be an object',
          severity: 'error',
        },
      ],
      parsed: null,
    };
  }

  // Layer 2: structural (Zod) validation.
  const result = topologyUISchemasSchema.safeParse(json);
  if (result.success) {
    return { diagnostics: [], parsed: json as TopologyUISchemas };
  }

  const diagnostics: Diagnostic[] = result.error.issues.map(
    (issue: ZodIssue) => {
      const node = nearestExistingNode(doc.contents, issue.path);
      let line = 1;
      let column = 1;
      if (node && node.range) {
        const start = offsetToLineCol(counter, node.range[0]);
        line = start.line;
        column = start.column;
      }
      return {
        line,
        column,
        message: messageForIssue(issue),
        severity: 'error' as const,
      };
    }
  );

  return { diagnostics, parsed: null };
};

// Turn a Zod issue into a human message, naming the missing/invalid key.
const messageForIssue = (issue: ZodIssue): string => {
  const key = issue.path[issue.path.length - 1];
  if (issue.code === 'invalid_union') {
    const where =
      issue.path.length > 0 ? `"${issue.path.join('.')}"` : 'this component';
    return `Invalid component at ${where}: each leaf needs a valid uiType (${FIELD_UI_TYPES.join(
      ' | '
    )}) or a group with a components map.`;
  }
  if (issue.code === 'invalid_enum_value') {
    return `Invalid value for "${String(key)}": ${issue.message}`;
  }
  if (
    issue.code === 'invalid_type' &&
    'received' in issue &&
    issue.received === 'undefined'
  ) {
    return `Missing required key "${String(key)}".`;
  }
  return `${String(key)}: ${issue.message}`;
};
