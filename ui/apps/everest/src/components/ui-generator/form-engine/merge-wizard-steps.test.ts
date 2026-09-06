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
import { StepDefinition } from './types';
import { mergeWizardSteps } from './merge-wizard-steps';

const createDummyStep = (id: string, sectionKey?: string): StepDefinition => ({
  id,
  label: id,
  sectionKey,
  component: () => null,
  fields: [],
});

describe('mergeWizardSteps', () => {
  const baseStep = createDummyStep('base');
  const importStep = createDummyStep('import');
  const backupStep = createDummyStep('backups');

  const standardStaticSteps = [baseStep, importStep, backupStep];

  const buildGeneratedStepsMap = (
    keys: string[]
  ): Map<string, StepDefinition> => {
    const map = new Map<string, StepDefinition>();
    for (const key of keys) {
      map.set(key, createDummyStep(`section:${key}`, key));
    }
    return map;
  };

  it('1. should keep default order when sectionsOrder is undefined (backward compat)', () => {
    const generatedSteps = buildGeneratedStepsMap(['resources', 'advanced']);

    const result = mergeWizardSteps(standardStaticSteps, generatedSteps);

    expect(result.map((s) => s.id)).toEqual([
      'base',
      'import',
      'backups',
      'section:resources',
      'section:advanced',
    ]);
  });

  it('2. should keep default static positions when sectionsOrder lists only schema keys', () => {
    const generatedSteps = buildGeneratedStepsMap(['resources', 'advanced']);

    const result = mergeWizardSteps(
      standardStaticSteps,
      generatedSteps,
      ['resources', 'advanced']
    );

    expect(result.map((s) => s.id)).toEqual([
      'base',
      'import',
      'backups',
      'section:resources',
      'section:advanced',
    ]);
  });

  it('3. should move backups to the end when specified in sectionsOrder', () => {
    const generatedSteps = buildGeneratedStepsMap(['resources', 'advanced']);

    const result = mergeWizardSteps(
      standardStaticSteps,
      generatedSteps,
      ['resources', 'advanced', 'backups']
    );

    expect(result.map((s) => s.id)).toEqual([
      'base',
      'import',
      'section:resources',
      'section:advanced',
      'backups',
    ]);
  });

  it('4. should place backups between schema sections when specified in sectionsOrder', () => {
    const generatedSteps = buildGeneratedStepsMap(['resources', 'advanced']);

    const result = mergeWizardSteps(
      standardStaticSteps,
      generatedSteps,
      ['resources', 'backups', 'advanced']
    );

    expect(result.map((s) => s.id)).toEqual([
      'base',
      'import',
      'section:resources',
      'backups',
      'section:advanced',
    ]);
  });

  it('5. should keep base pinned at index 0 even if listed later in sectionsOrder', () => {
    const generatedSteps = buildGeneratedStepsMap(['resources', 'advanced']);

    const result = mergeWizardSteps(
      standardStaticSteps,
      generatedSteps,
      ['base', 'resources', 'backups']
    );

    expect(result.map((s) => s.id)).toEqual([
      'base',
      'import',
      'section:resources',
      'backups',
      'section:advanced',
    ]);
  });

  it('6. should ignore unknown keys in sectionsOrder', () => {
    const generatedSteps = buildGeneratedStepsMap(['resources', 'advanced']);

    const result = mergeWizardSteps(
      standardStaticSteps,
      generatedSteps,
      ['resources', 'nonexistent', 'advanced']
    );

    expect(result.map((s) => s.id)).toEqual([
      'base',
      'import',
      'backups',
      'section:resources',
      'section:advanced',
    ]);
  });

  it('7. should order correctly when no static steps except base exist', () => {
    const generatedSteps = buildGeneratedStepsMap(['resources', 'advanced']);

    const result = mergeWizardSteps(
      [baseStep],
      generatedSteps,
      ['resources', 'advanced']
    );

    expect(result.map((s) => s.id)).toEqual([
      'base',
      'section:resources',
      'section:advanced',
    ]);
  });

  it('8. should append unlisted generated steps at the end for partial sectionsOrder', () => {
    const generatedSteps = buildGeneratedStepsMap(['resources', 'advanced']);

    const result = mergeWizardSteps(
      standardStaticSteps,
      generatedSteps,
      ['advanced']
    );

    expect(result.map((s) => s.id)).toEqual([
      'base',
      'import',
      'backups',
      'section:advanced',
      'section:resources',
    ]);
  });

  it('9. should drop generated section on collision with backups when explicit in sectionsOrder (fixture: base + backups, no import)', () => {
    // Schema defines a section keyed 'backups' in addition to resources
    const generatedSteps = buildGeneratedStepsMap(['backups', 'resources']);
    const staticSteps = [baseStep, backupStep]; // no import step in this fixture

    const result = mergeWizardSteps(
      staticSteps,
      generatedSteps,
      ['backups', 'resources']
    );

    // Only the static backups step is present; generated section:backups is dropped
    expect(result.map((s) => s.id)).toEqual([
      'base',
      'backups',
      'section:resources',
    ]);
  });

  it('10. should drop generated section on collision with backups when sectionsOrder is undefined (fixture: base + backups, no import)', () => {
    // Schema defines a section keyed 'backups' in addition to resources and advanced
    const generatedSteps = buildGeneratedStepsMap([
      'backups',
      'resources',
      'advanced',
    ]);
    const staticSteps = [baseStep, backupStep]; // no import step in this fixture

    const result = mergeWizardSteps(staticSteps, generatedSteps);

    // Only the static backups step is present; generated section:backups is dropped
    expect(result.map((s) => s.id)).toEqual([
      'base',
      'backups',
      'section:resources',
      'section:advanced',
    ]);
  });

  it('11. should drop generated section on collision with import (fixture: base + import + backups)', () => {
    // Schema defines a section keyed 'import' in addition to resources
    const generatedSteps = buildGeneratedStepsMap(['import', 'resources']);

    const result = mergeWizardSteps(
      standardStaticSteps,
      generatedSteps,
      ['import', 'resources']
    );

    // Static backups is unlisted so it keeps its default position before generated sections;
    // Static import is listed and rendered; generated section:import is dropped
    expect(result.map((s) => s.id)).toEqual([
      'base',
      'backups',
      'import',
      'section:resources',
    ]);
  });
});
