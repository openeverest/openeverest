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

import { PitrStorageStatus } from './restore-pitr.types';
import { getPitrWindow, toRestoreDateISO } from './restore-pitr.utils';

describe('toRestoreDateISO', () => {
  it('emits an offset-carrying UTC timestamp without milliseconds', () => {
    expect(toRestoreDateISO(new Date('2026-07-29T10:15:00.123Z'))).toBe(
      '2026-07-29T10:15:00Z'
    );
  });

  it('converts a zoned instant to the correct UTC point', () => {
    // 12:15 at +02:00 is 10:15 UTC — the offset must not be dropped or mislabeled.
    expect(toRestoreDateISO(new Date('2026-07-29T12:15:00+02:00'))).toBe(
      '2026-07-29T10:15:00Z'
    );
  });
});

describe('getPitrWindow', () => {
  it('returns a usable window when Available with both bounds', () => {
    const status: PitrStorageStatus = {
      state: 'Available',
      earliestRestorableTime: '2026-07-29T00:00:00Z',
      latestRestorableTime: '2026-07-29T12:00:00Z',
      message: 'ok',
    };

    expect(getPitrWindow(status)).toEqual({
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

    expect(getPitrWindow(status)).toEqual({
      available: false,
      message: 'no successful backup yet',
    });
  });

  it('is unavailable when Available but a bound is missing', () => {
    const status: PitrStorageStatus = {
      state: 'Available',
      latestRestorableTime: '2026-07-29T12:00:00Z',
    };

    expect(getPitrWindow(status)).toEqual({ available: false });
  });

  it('is unavailable when no status is reported', () => {
    expect(getPitrWindow(undefined)).toEqual({ available: false });
  });
});
