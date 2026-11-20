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

import {
  getOpenApiConstraintError,
  getOpenApiConstraints,
} from './openapi-constraints';
import { Messages } from './openapi-constraints.messages';

const schema = {
  type: 'object',
  required: ['endpoint', 'auth'],
  properties: {
    endpoint: {
      type: 'string',
      pattern: '^https?://[^/?#@]+([/?#].*)?$',
      maxLength: 2048,
    },
    region: { type: 'string', minLength: 1, maxLength: 63 },
    implementation: { type: 'string', enum: ['aws', 'minio'] },
    auth: {
      type: 'object',
      required: ['user'],
      properties: { user: { type: 'string' }, realm: { type: 'string' } },
    },
    tls: {
      type: 'object',
      required: ['ca'],
      properties: { ca: { type: 'string' } },
    },
    labels: {
      type: 'object',
      additionalProperties: { type: 'string', pattern: '^[a-z]+$' },
    },
  },
};

describe('getOpenApiConstraints', () => {
  it('requires a value only when every object on the way requires the next key', () => {
    expect(getOpenApiConstraints(schema, ['auth', 'user'])).toEqual({
      required: true,
    });
    expect(getOpenApiConstraints(schema, ['auth', 'realm'])).toEqual({
      required: false,
    });
    // tls itself is optional, so its required ca only applies once tls is set.
    expect(getOpenApiConstraints(schema, ['tls', 'ca'])).toEqual({
      required: false,
    });
  });

  it('reads the constraints of a map entry from additionalProperties', () => {
    expect(getOpenApiConstraints(schema, ['labels', 'team'])).toEqual({
      required: false,
      pattern: '^[a-z]+$',
    });
  });

  it("knows nothing of a path the schema doesn't describe", () => {
    expect(getOpenApiConstraints(schema, ['proxy'])).toBeUndefined();
    expect(getOpenApiConstraints(schema, ['region', 'code'])).toBeUndefined();
    expect(getOpenApiConstraints(undefined, ['region'])).toBeUndefined();
  });
});

describe('getOpenApiConstraintError', () => {
  const errorAt = (path: string[], value: unknown) => {
    const constraints = getOpenApiConstraints(schema, path);
    return constraints && getOpenApiConstraintError(value, constraints);
  };

  it('asks for a required value and lets an optional one be empty', () => {
    expect(errorAt(['endpoint'], '')).toBe(Messages.required);
    expect(errorAt(['endpoint'], undefined)).toBe(Messages.required);
    expect(errorAt(['region'], undefined)).toBeUndefined();
  });

  it('checks the pattern, lengths and allowed values', () => {
    expect(
      errorAt(['endpoint'], 'https://s3.example.com/path')
    ).toBeUndefined();
    expect(errorAt(['endpoint'], 'https://user@s3.example.com')).toBe(
      Messages.pattern
    );
    expect(errorAt(['region'], 'r'.repeat(64))).toBe(Messages.maxLength(63));
    expect(errorAt(['implementation'], 'ceph')).toBe(
      Messages.oneOf(['aws', 'minio'])
    );
  });

  it('counts characters as the API does, not UTF-16 code units', () => {
    expect(
      getOpenApiConstraintError('🙂🙂', { required: false, maxLength: 2 })
    ).toBeUndefined();
  });

  it('reads Unicode property escapes in a pattern', () => {
    const constraints = { required: false, pattern: '^\\p{L}+$' };

    expect(getOpenApiConstraintError('Zürich', constraints)).toBeUndefined();
    expect(getOpenApiConstraintError('Zürich 1', constraints)).toBe(
      Messages.pattern
    );
  });

  it('reads an escaped punctuation character as the character, as Go does', () => {
    const constraints = { required: false, pattern: '^a\\_b\\-c$' };

    expect(getOpenApiConstraintError('a_b-c', constraints)).toBeUndefined();
    expect(getOpenApiConstraintError('abc', constraints)).toBe(
      Messages.pattern
    );
  });

  it('keeps the escape of a character with a meaning, and of "-" in a class', () => {
    const constraints = { required: false, pattern: '^[a\\-z]\\.\\_$' };

    expect(getOpenApiConstraintError('-._', constraints)).toBeUndefined();
    // Not the range a-z.
    expect(getOpenApiConstraintError('b._', constraints)).toBe(
      Messages.pattern
    );
    // Not any character.
    expect(getOpenApiConstraintError('-x_', constraints)).toBe(
      Messages.pattern
    );
  });

  it('leaves a pattern only Go reads to the server', () => {
    expect(
      getOpenApiConstraintError('v1', {
        required: false,
        pattern: '^v[0-9]+\\z',
      })
    ).toBeUndefined();
    expect(
      getOpenApiConstraintError('abc', {
        required: false,
        pattern: '^[[:alnum:]]+$',
      })
    ).toBeUndefined();
  });
});
