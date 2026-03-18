import { describe, expect, it } from 'vitest';
import { FieldType } from '../../ui-generator.types';
import {
  getComponentSourcePath,
  getComponentTargetPaths,
  withNormalizedPathMeta,
} from './normalized-component';

describe('normalized-component', () => {
  it('uses first path as source and keeps all targets for multipath fields', () => {
    const component = withNormalizedPathMeta({
      uiType: FieldType.Text,
      path: ['spec.engine.version', 'spec.proxy.version'],
      fieldParams: { label: 'Version' },
    });

    expect(getComponentSourcePath(component)).toBe('spec.engine.version');
    expect(getComponentTargetPaths(component)).toEqual([
      'spec.engine.version',
      'spec.proxy.version',
    ]);
  });

  it('supports single path fields', () => {
    const component = withNormalizedPathMeta({
      uiType: FieldType.Number,
      path: 'spec.replicas',
      fieldParams: { label: 'Replicas' },
    });

    expect(getComponentSourcePath(component)).toBe('spec.replicas');
    expect(getComponentTargetPaths(component)).toEqual(['spec.replicas']);
  });

  it('returns no source path for id-only fields', () => {
    const component = withNormalizedPathMeta({
      uiType: FieldType.Hidden,
      id: 'ui.hidden-field',
      fieldParams: {},
    });

    expect(getComponentSourcePath(component)).toBeUndefined();
    expect(getComponentTargetPaths(component)).toEqual([]);
  });
});
