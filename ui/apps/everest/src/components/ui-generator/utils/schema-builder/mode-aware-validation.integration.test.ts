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
import { buildSectionZodSchema } from './build-section-zod-schema';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import {
  genericOriginalInstance,
  pickFixtureSections,
} from 'components/ui-generator/test-fixtures/generic-topology.fixture';

//  TODO should be moved to e2e tests

const genericSections = pickFixtureSections(['resources']);

describe('mode-aware validation integration', () => {
  describe('new mode — no edit constraints', () => {
    it('allows setting nodes to 1 (no descaling check)', () => {
      const { schema } = buildSectionZodSchema('resources', genericSections, {
        formMode: FormMode.New,
      });

      const result = schema.safeParse({
        spec: {
          components: {
            engine: { replicas: 1, storage: { size: 5 } },
            configServer: { replicas: 1 },
          },
          sharding: { shards: 1 },
        },
      });

      expect(result.success).toBe(true);
    });

    it('still enforces base odd-node CEL in new mode', () => {
      const { schema } = buildSectionZodSchema('resources', genericSections, {
        formMode: FormMode.New,
      });

      const result = schema.safeParse({
        spec: {
          components: {
            engine: { replicas: 2, storage: { size: 5 } },
            configServer: { replicas: 3 },
          },
          sharding: { shards: 1 },
        },
      });

      expect(result.success).toBe(false);
      const messages = result.error!.issues.map((i) => i.message);
      expect(messages).toContain('The number of nodes must be odd');
    });
  });

  describe('edit mode — disk descaling blocked', () => {
    it('rejects decreasing disk size below original', () => {
      const { schema } = buildSectionZodSchema('resources', genericSections, {
        formMode: FormMode.Edit,
        originalData: genericOriginalInstance,
      });

      const result = schema.safeParse({
        spec: {
          components: {
            engine: { replicas: 3, storage: { size: 10 } },
            configServer: { replicas: 3 },
          },
          sharding: { shards: 2 },
        },
      });

      expect(result.success).toBe(false);
      const messages = result.error!.issues.map((i) => i.message);
      expect(messages).toContain('Disk size cannot be decreased');
    });

    it('allows increasing disk size above original', () => {
      const { schema } = buildSectionZodSchema('resources', genericSections, {
        formMode: FormMode.Edit,
        originalData: genericOriginalInstance,
      });

      const result = schema.safeParse({
        spec: {
          components: {
            engine: { replicas: 3, storage: { size: 50 } },
            configServer: { replicas: 3 },
          },
          sharding: { shards: 2 },
        },
      });

      expect(result.success).toBe(true);
    });
  });

  describe('edit mode — shard descaling blocked', () => {
    it('rejects decreasing shards below original', () => {
      const { schema } = buildSectionZodSchema('resources', genericSections, {
        formMode: FormMode.Edit,
        originalData: genericOriginalInstance,
      });

      const result = schema.safeParse({
        spec: {
          components: {
            engine: { replicas: 3, storage: { size: 25 } },
            configServer: { replicas: 3 },
          },
          sharding: { shards: 1 },
        },
      });

      expect(result.success).toBe(false);
      const messages = result.error!.issues.map((i) => i.message);
      expect(messages).toContain('Number of shards cannot be decreased');
    });

    it('allows increasing shards above original', () => {
      const { schema } = buildSectionZodSchema('resources', genericSections, {
        formMode: FormMode.Edit,
        originalData: genericOriginalInstance,
      });

      const result = schema.safeParse({
        spec: {
          components: {
            engine: { replicas: 3, storage: { size: 25 } },
            configServer: { replicas: 3 },
          },
          sharding: { shards: 4 },
        },
      });

      expect(result.success).toBe(true);
    });
  });

  describe('edit mode — node >1 to 1 blocked', () => {
    it('rejects scaling nodes from >1 down to 1', () => {
      const { schema } = buildSectionZodSchema('resources', genericSections, {
        formMode: FormMode.Edit,
        originalData: genericOriginalInstance,
      });

      const result = schema.safeParse({
        spec: {
          components: {
            engine: { replicas: 1, storage: { size: 25 } },
            configServer: { replicas: 3 },
          },
          sharding: { shards: 2 },
        },
      });

      expect(result.success).toBe(false);
      const messages = result.error!.issues.map((i) => i.message);
      expect(messages).toContain('Cannot scale down to a single node');
    });

    it('allows changing from 3 to 5 nodes', () => {
      const { schema } = buildSectionZodSchema('resources', genericSections, {
        formMode: FormMode.Edit,
        originalData: genericOriginalInstance,
      });

      const result = schema.safeParse({
        spec: {
          components: {
            engine: { replicas: 5, storage: { size: 25 } },
            configServer: { replicas: 3 },
          },
          sharding: { shards: 2 },
        },
      });

      expect(result.success).toBe(true);
    });
  });

  describe('edit mode — base + edit CEL combined on nodes', () => {
    it('rejects even node count in edit mode too (base CEL)', () => {
      const { schema } = buildSectionZodSchema('resources', genericSections, {
        formMode: FormMode.Edit,
        originalData: genericOriginalInstance,
      });

      const result = schema.safeParse({
        spec: {
          components: {
            engine: { replicas: 4, storage: { size: 25 } },
            configServer: { replicas: 3 },
          },
          sharding: { shards: 2 },
        },
      });

      expect(result.success).toBe(false);
      const messages = result.error!.issues.map((i) => i.message);
      expect(messages).toContain('The number of nodes must be odd');
    });
  });

  describe('base config servers CEL — works in all modes', () => {
    it('rejects 1 configServer when nodes > 1 in new mode', () => {
      const { schema } = buildSectionZodSchema('resources', genericSections, {
        formMode: FormMode.New,
      });

      const result = schema.safeParse({
        spec: {
          components: {
            engine: { replicas: 3, storage: { size: 25 } },
            configServer: { replicas: 1 },
          },
          sharding: { shards: 1 },
        },
      });

      expect(result.success).toBe(false);
      const messages = result.error!.issues.map((i) => i.message);
      expect(messages).toContain(
        'Config servers cannot be 1 if the number of nodes is greater than 1'
      );
    });

    it('allows 1 configServer when nodes == 1 in new mode', () => {
      const { schema } = buildSectionZodSchema('resources', genericSections, {
        formMode: FormMode.New,
      });

      const result = schema.safeParse({
        spec: {
          components: {
            engine: { replicas: 1, storage: { size: 25 } },
            configServer: { replicas: 1 },
          },
          sharding: { shards: 1 },
        },
      });

      expect(result.success).toBe(true);
    });
  });

  describe('no formMode — only base rules apply', () => {
    it('ignores edit-only CEL when formMode is not provided', () => {
      const { schema } = buildSectionZodSchema('resources', genericSections);

      // Disk decreased (would fail in edit), but no formMode => no edit CEL
      const result = schema.safeParse({
        spec: {
          components: {
            engine: { replicas: 3, storage: { size: 5 } },
            configServer: { replicas: 3 },
          },
          sharding: { shards: 1 },
        },
      });

      expect(result.success).toBe(true);
    });
  });
});
