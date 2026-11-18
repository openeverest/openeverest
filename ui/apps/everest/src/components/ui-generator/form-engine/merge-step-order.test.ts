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
import { mergeStepOrder, OrderableStep } from './merge-step-order';

const step = (id: string, sectionKey?: string): OrderableStep => ({
  id,
  sectionKey,
});

describe('mergeStepOrder', () => {
  it('preserves the existing section-key-only ordering', () => {
    const staticSteps = [step('base'), step('backups')];
    const generatedSteps = [
      step('section:resources', 'resources'),
      step('section:advanced', 'advanced'),
    ];

    expect(
      mergeStepOrder(staticSteps, generatedSteps, [
        'resources',
        'advanced',
      ]).map(({ id }) => id)
    ).toEqual(['base', 'backups', 'section:resources', 'section:advanced']);
  });

  it('interleaves static IDs with section keys', () => {
    const staticSteps = [step('base'), step('backups')];
    const generatedSteps = [
      step('section:resources', 'resources'),
      step('section:advanced', 'advanced'),
    ];

    expect(
      mergeStepOrder(staticSteps, generatedSteps, [
        'resources',
        'advanced',
        'backups',
      ]).map(({ id }) => id)
    ).toEqual(['base', 'section:resources', 'section:advanced', 'backups']);
  });

  it('places a static ID between two schema sections', () => {
    const staticSteps = [step('base'), step('backups')];
    const generatedSteps = [
      step('section:resources', 'resources'),
      step('section:advanced', 'advanced'),
    ];

    expect(
      mergeStepOrder(staticSteps, generatedSteps, [
        'resources',
        'backups',
        'advanced',
      ]).map(({ id }) => id)
    ).toEqual(['base', 'section:resources', 'backups', 'section:advanced']);
  });

  it('keeps unlisted static steps before generated sections', () => {
    const staticSteps = [step('base'), step('monitoring'), step('backups')];
    const generatedSteps = [
      step('section:resources', 'resources'),
      step('section:advanced', 'advanced'),
    ];

    expect(
      mergeStepOrder(staticSteps, generatedSteps, ['resources']).map(
        ({ id }) => id
      )
    ).toEqual([
      'base',
      'monitoring',
      'backups',
      'section:resources',
      'section:advanced',
    ]);
  });

  it('always pins base first even if it is listed elsewhere', () => {
    const staticSteps = [step('backups'), step('base')];
    const generatedSteps = [step('section:resources', 'resources')];

    expect(
      mergeStepOrder(staticSteps, generatedSteps, [
        'resources',
        'backups',
        'base',
      ]).map(({ id }) => id)
    ).toEqual(['base', 'section:resources', 'backups']);
  });

  it('gives a static step precedence over a colliding schema section', () => {
    const staticSteps = [step('base'), step('backups')];
    const generatedSteps = [
      step('section:backups', 'backups'),
      step('section:resources', 'resources'),
    ];

    expect(
      mergeStepOrder(staticSteps, generatedSteps, ['backups', 'resources']).map(
        ({ id }) => id
      )
    ).toEqual(['base', 'backups', 'section:resources']);
  });

  it('ignores unknown keys in sectionsOrder', () => {
    const staticSteps = [step('base'), step('backups')];
    const generatedSteps = [step('section:resources', 'resources')];

    expect(
      mergeStepOrder(staticSteps, generatedSteps, [
        'unknown',
        'resources',
        'missing',
      ]).map(({ id }) => id)
    ).toEqual(['base', 'backups', 'section:resources']);
  });

  it('gives a static step precedence over a collision in fallback order', () => {
    const staticSteps = [step('base'), step('backups')];
    const generatedSteps = [
      step('section:backups', 'backups'),
      step('section:resources', 'resources'),
    ];

    expect(
      mergeStepOrder(staticSteps, generatedSteps, ['resources']).map(
        ({ id }) => id
      )
    ).toEqual(['base', 'backups', 'section:resources']);
  });
});
