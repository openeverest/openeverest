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

import { isEmptyFieldValue, isPlainObject } from '../object-path';
import { Messages } from './openapi-constraints.messages';
import { OpenApiConstraints } from './openapi-constraints.types';

const getChildSchema = (
  schema: Record<string, unknown>,
  key: string
): { schema: unknown; required: boolean } | undefined => {
  const { properties, additionalProperties } = schema;
  if (
    isPlainObject(properties) &&
    Object.prototype.hasOwnProperty.call(properties, key)
  ) {
    return {
      schema: properties[key],
      required: Array.isArray(schema.required) && schema.required.includes(key),
    };
  }
  // A map's entries: any key, none of them required.
  if (isPlainObject(additionalProperties)) {
    return { schema: additionalProperties, required: false };
  }
  return undefined;
};

// Undefined for a path the schema doesn't describe. anyOf/oneOf/allOf are not
// followed: what they ask is left to the server.
export const getOpenApiConstraints = (
  schema: unknown,
  path: string[]
): OpenApiConstraints | undefined => {
  if (path.length === 0) {
    return undefined;
  }
  let node = schema;
  let required = true;
  for (const key of path) {
    const child = isPlainObject(node) ? getChildSchema(node, key) : undefined;
    if (!child) {
      return undefined;
    }
    node = child.schema;
    required = required && child.required;
  }
  if (!isPlainObject(node)) {
    return undefined;
  }
  const { pattern, minLength, maxLength } = node;
  return {
    required,
    ...(typeof pattern === 'string' && { pattern }),
    ...(typeof minLength === 'number' && { minLength }),
    ...(typeof maxLength === 'number' && { maxLength }),
    ...(Array.isArray(node.enum) && { enum: [...node.enum] }),
  };
};

// The ASCII characters other than letters and digits, which Go reads escaped
// as the character itself (\_ is _).
const GO_LITERAL_ESCAPE = /^[ -/:-@[-`{-~]$/;
// Those JavaScript's Unicode mode lets a pattern escape: its syntax
// characters and "/", and "-" in a class.
const UNICODE_MODE_ESCAPES = new Set([...'^$\\.*+?()[]{}|/']);

// The escapes of characters without a meaning lose their backslash, which the
// Unicode mode refuses. A "-" in a class keeps it: bare, it makes a range.
const unescapeLiteralPunctuation = (pattern: string): string => {
  let inClass = false;
  return pattern.replace(
    /\\([\s\S])|[[\]]/g,
    (match: string, escaped?: string) => {
      if (escaped === undefined) {
        inClass = match === '[';
        return match;
      }
      const keepsEscape =
        !GO_LITERAL_ESCAPE.test(escaped) ||
        UNICODE_MODE_ESCAPES.has(escaped) ||
        (escaped === '-' && inClass);
      return keepsEscape ? match : escaped;
    }
  );
};

// The server checks a pattern with Go's regexp. JavaScript's Unicode mode
// reads it the same way once escaped punctuation is unescaped; a pattern only
// Go reads, such as one with \z or [[:alpha:]], is left to the server.
const toRegExp = (pattern: string): RegExp | undefined => {
  try {
    return new RegExp(unescapeLiteralPunctuation(pattern), 'u');
  } catch {
    return undefined;
  }
};

// The API counts characters, where String.length counts UTF-16 code units.
const countCharacters = (value: string) => [...value].length;

export const getOpenApiConstraintError = (
  value: unknown,
  { required, pattern, minLength, maxLength, enum: allowed }: OpenApiConstraints
): string | undefined => {
  if (isEmptyFieldValue(value)) {
    return required ? Messages.required : undefined;
  }
  if (allowed && !allowed.includes(value)) {
    return Messages.oneOf(allowed);
  }
  if (typeof value !== 'string') {
    return undefined;
  }
  const length = countCharacters(value);
  if (minLength !== undefined && length < minLength) {
    return Messages.minLength(minLength);
  }
  if (maxLength !== undefined && length > maxLength) {
    return Messages.maxLength(maxLength);
  }
  const regExp = pattern === undefined ? undefined : toRegExp(pattern);
  return regExp && !regExp.test(value) ? Messages.pattern : undefined;
};
