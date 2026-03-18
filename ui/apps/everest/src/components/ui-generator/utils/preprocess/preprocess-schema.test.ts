import { describe, expect, it } from 'vitest';
import { Component, FieldType, TopologyUISchemas } from '../../ui-generator.types';
import { preprocessSchema } from './preprocess-schema';

describe('preprocessSchema', () => {
  it('normalizes path metadata even without provider object', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          base: {
            components: {
              version: {
                uiType: FieldType.Text,
                path: ['spec.engine.version', 'spec.proxy.version'],
                fieldParams: { label: 'Version' },
              },
            },
          },
        },
      },
    };

    const result = preprocessSchema(schema);
    const component = result.single.sections.base.components.version;

    if ('uiType' in component && !('components' in component)) {
      const leaf = component as Component;
      expect(leaf._normalized?.sourcePath).toBe('spec.engine.version');
      expect(leaf._normalized?.targetPaths).toEqual([
        'spec.engine.version',
        'spec.proxy.version',
      ]);
    }
  });
});
