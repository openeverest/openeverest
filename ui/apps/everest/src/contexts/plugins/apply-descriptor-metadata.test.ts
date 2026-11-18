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

import { describe, expect, it } from 'vitest';
import type { Extension } from '@openeverest/plugin-sdk';
import {
  applyDescriptorMetadata,
  extensionAppliesToProvider,
  type ExtensionPointDescriptor,
} from './apply-descriptor-metadata';

const NoopComponent = () => null;

describe('applyDescriptorMetadata', () => {
  it('applies the CR provider filter when the bundle omitted it', () => {
    const ext: Extension = {
      type: 'clusterDetailTab',
      label: 'MongoDB Explorer',
      path: 'mongodb-explorer',
      component: NoopComponent,
    };
    const descriptors: ExtensionPointDescriptor[] = [
      {
        type: 'clusterDetailTab',
        path: 'mongodb-explorer',
        providers: ['percona-server-mongodb'],
      },
    ];

    applyDescriptorMetadata(ext, descriptors);

    expect(ext.providers).toEqual(['percona-server-mongodb']);
  });

  it('lets the CR override a provider filter hardcoded in the bundle', () => {
    const ext: Extension = {
      type: 'clusterDetailTab',
      label: 'Tab',
      path: 'tab',
      component: NoopComponent,
      providers: ['provider-percona-postgresql'],
    };
    const descriptors: ExtensionPointDescriptor[] = [
      {
        type: 'clusterDetailTab',
        path: 'tab',
        providers: ['percona-server-mongodb'],
      },
    ];

    applyDescriptorMetadata(ext, descriptors);

    expect(ext.providers).toEqual(['percona-server-mongodb']);
  });

  it('matches by type+path so same-typed extensions are not confused', () => {
    const ext: Extension = {
      type: 'clusterDetailTab',
      label: 'Second',
      path: 'second',
      component: NoopComponent,
    };
    const descriptors: ExtensionPointDescriptor[] = [
      { type: 'clusterDetailTab', path: 'first', providers: ['a'] },
      { type: 'clusterDetailTab', path: 'second', providers: ['b'] },
    ];

    applyDescriptorMetadata(ext, descriptors);

    expect(ext.providers).toEqual(['b']);
  });

  it('does not touch providers when the CR entry omits them', () => {
    const ext: Extension = {
      type: 'clusterDetailTab',
      label: 'Tab',
      path: 'tab',
      component: NoopComponent,
      providers: ['provider-percona-postgresql'],
    };
    const descriptors: ExtensionPointDescriptor[] = [
      { type: 'clusterDetailTab', path: 'tab' },
    ];

    applyDescriptorMetadata(ext, descriptors);

    expect(ext.providers).toEqual(['provider-percona-postgresql']);
  });

  it('backfills a missing sidebar icon from the CR, matched by label', () => {
    const ext: Extension = { type: 'sidebarItem', label: 'MongoDB Explorer' };
    const descriptors: ExtensionPointDescriptor[] = [
      {
        type: 'sidebarItem',
        label: 'MongoDB Explorer',
        icon: '/v1/plugins/mongodb-explorer/icon.png',
      },
    ];

    applyDescriptorMetadata(ext, descriptors);

    expect(ext.icon).toBe('/v1/plugins/mongodb-explorer/icon.png');
  });

  it('does not overwrite an icon the bundle already set', () => {
    const ext: Extension = {
      type: 'sidebarItem',
      label: 'MongoDB Explorer',
      icon: '/bundle-icon.png',
    };
    const descriptors: ExtensionPointDescriptor[] = [
      { type: 'sidebarItem', label: 'MongoDB Explorer', icon: '/cr-icon.png' },
    ];

    applyDescriptorMetadata(ext, descriptors);

    expect(ext.icon).toBe('/bundle-icon.png');
  });
});

describe('applyDescriptorMetadata match result', () => {
  const descriptors: ExtensionPointDescriptor[] = [
    { type: 'clusterAction', label: 'Explore', providers: ['a'] },
  ];

  it('returns the matched CR entry and scopes a label-matched action', () => {
    const ext: Extension = {
      type: 'clusterAction',
      label: 'Explore',
      component: NoopComponent,
    };

    expect(applyDescriptorMetadata(ext, descriptors)).toBe(descriptors[0]);
    expect(ext.providers).toEqual(['a']);
  });

  it('returns undefined when the labels differ, leaving the action unscoped', () => {
    const ext: Extension = {
      type: 'clusterAction',
      label: 'Renamed',
      component: NoopComponent,
    };

    expect(applyDescriptorMetadata(ext, descriptors)).toBeUndefined();
    expect(ext.providers).toBeUndefined();
  });
});

describe('extensionAppliesToProvider', () => {
  it('applies to every provider without a filter', () => {
    expect(extensionAppliesToProvider({}, 'a')).toBe(true);
    expect(extensionAppliesToProvider({ providers: [] }, undefined)).toBe(true);
  });

  it('applies only to listed providers when filtered', () => {
    expect(extensionAppliesToProvider({ providers: ['a'] }, 'a')).toBe(true);
    expect(extensionAppliesToProvider({ providers: ['a'] }, 'b')).toBe(false);
    expect(extensionAppliesToProvider({ providers: ['a'] }, undefined)).toBe(
      false
    );
  });
});
