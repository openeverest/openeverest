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

import { isSubmitDisabled } from './database-form-body.utils';

describe('isSubmitDisabled', () => {
  it('blocks a picked preset until it resolves, ignoring form validity', () => {
    expect(
      isSubmitDisabled({
        isSubmitting: false,
        presetPending: true,
        presetSelected: true,
        isValid: true,
      })
    ).toBe(true);
  });

  it('allows a one-click preset deploy even when the form is invalid', () => {
    // Preset resolved: the server validates the spec, so form validity must not
    // gate the deploy.
    expect(
      isSubmitDisabled({
        isSubmitting: false,
        presetPending: false,
        presetSelected: true,
        isValid: false,
      })
    ).toBe(false);
  });

  it('still requires a valid form for manual (no-preset) creation', () => {
    expect(
      isSubmitDisabled({
        isSubmitting: false,
        presetPending: false,
        presetSelected: false,
        isValid: false,
      })
    ).toBe(true);

    expect(
      isSubmitDisabled({
        isSubmitting: false,
        presetPending: false,
        presetSelected: false,
        isValid: true,
      })
    ).toBe(false);
  });

  it('always blocks while submitting', () => {
    expect(
      isSubmitDisabled({
        isSubmitting: true,
        presetPending: false,
        presetSelected: true,
        isValid: true,
      })
    ).toBe(true);
  });
});
