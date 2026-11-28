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

import type {
  Component,
  ComponentGroup,
} from 'components/ui-generator/ui-generator.types';
import { getComponentSourcePath } from '../preprocess/normalized-component';
import { toFormId } from 'components/ui-generator/utils/object-path';

export type GeneratedName = string | readonly string[];

// Strings retain legacy dotted-hierarchy semantics; arrays preserve literal dots.
export const getGeneratedNameSegments = (
  generatedName: GeneratedName
): string[] =>
  typeof generatedName === 'string'
    ? generatedName.length > 0
      ? generatedName.split('.')
      : []
    : [...generatedName];

const RESERVED_OBJECT_SEGMENTS = new Set([
  '__proto__',
  'constructor',
  'prototype',
]);

const encodeGeneratedSegment = (segment: string): string => {
  if (segment.length === 0) return '%';

  // Escape within segments so encoded data cannot become RHF path syntax.
  const encoded = segment.replace(/[^A-Za-z0-9_-]/gu, (character) => {
    try {
      return encodeURIComponent(character).replace(
        /[.!'()*~]/g,
        (unsafe) => `%${unsafe.charCodeAt(0).toString(16).toUpperCase()}`
      );
    } catch (error) {
      if (!(error instanceof URIError)) throw error;
      return `%u${character
        .charCodeAt(0)
        .toString(16)
        .toUpperCase()
        .padStart(4, '0')}`;
    }
  });

  if (/^\d+$/u.test(segment) || RESERVED_OBJECT_SEGMENTS.has(segment)) {
    const firstCharacter = segment.charCodeAt(0).toString(16).toUpperCase();
    return `%${firstCharacter.padStart(2, '0')}${encoded.slice(1)}`;
  }

  return encoded;
};

// Uses the 'path' property if available, otherwise generates an ID from the component name.
// The id is the React Hook Form field name, so it must never contain the
// canonical `['a.b']` syntax (RHF would split it) — see toFormId.
export const generateFieldId = (
  item: Component | ComponentGroup,
  generatedName: GeneratedName
): string => {
  const sourcePath = getComponentSourcePath(item);
  if (sourcePath) {
    return toFormId(sourcePath);
  }

  const segments = getGeneratedNameSegments(generatedName);
  return `g-${segments.map(encodeGeneratedSegment).join('.')}`;
};
