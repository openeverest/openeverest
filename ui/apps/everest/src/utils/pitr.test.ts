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

import { PitrStorageStatus } from 'shared-types/pitr.types';
import { resolvePitrWindow } from './pitr';

describe('resolvePitrWindow', () => {
  it('returns a usable window when Available with both bounds', () => {
    const status: PitrStorageStatus = {
      state: 'Available',
      earliestRestorableTime: '2026-07-29T00:00:00Z',
      latestRestorableTime: '2026-07-29T12:00:00Z',
      message: 'ok',
    };

    expect(resolvePitrWindow(status)).toEqual({
      available: true,
      earliest: new Date('2026-07-29T00:00:00Z'),
      latest: new Date('2026-07-29T12:00:00Z'),
      message: 'ok',
    });
  });

  it('is unavailable (with the message) when the state is Unavailable', () => {
    const status: PitrStorageStatus = {
      state: 'Unavailable',
      message: 'no successful backup yet',
    };

    expect(resolvePitrWindow(status)).toEqual({
      available: false,
      message: 'no successful backup yet',
    });
  });

  it('is unavailable when Available but a bound is missing', () => {
    const status: PitrStorageStatus = {
      state: 'Available',
      latestRestorableTime: '2026-07-29T12:00:00Z',
    };

    expect(resolvePitrWindow(status)).toEqual({ available: false });
  });

  it('is unavailable when no status is reported', () => {
    expect(resolvePitrWindow(undefined)).toEqual({ available: false });
  });
});
