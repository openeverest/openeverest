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

// Host-version handling for the plugin compatibility check, kept out of the
// context module so it can be unit tested without pulling in the provider.

/**
 * Everest reports versions with a leading `v` and an optional build suffix
 * (e.g. "v0.3.0-1-a93bef"). Returns null when the value isn't a usable version
 * — including the "dev" fallback and the unrendered Go template placeholder
 * that `vite dev` serves, since it doesn't run the server-side template.
 */
export function parseVersion(value: string): [number, number, number] | null {
  const m = value
    .trim()
    .replace(/^v/i, '')
    .match(/^(\d+)\.(\d+)\.(\d+)/);
  return m ? [Number(m[1]), Number(m[2]), Number(m[3])] : null;
}

type Version = [number, number, number];

const compareVersions = (a: Version, b: Version): number =>
  a[0] - b[0] || a[1] - b[1] || a[2] - b[2];

// Evaluates a semver range (npm syntax: ">=2.0.0 <3.0.0", "^2.1.0", "~2.1.0",
// "1.x || 2.x") against an already-parsed version. Unknown comparators pass so a
// malformed range never silently blocks a plugin (advisory, first-party).
function satisfiesRange(host: Version, range: string): boolean {
  const evalClause = (clause: string): boolean => {
    const comparators = clause.trim().split(/\s+/).filter(Boolean);
    if (comparators.length === 0) {
      return true;
    }
    return comparators.every((c) => {
      const m = c.match(/^(>=|<=|>|<|=|\^|~)?(v?\d+\.\d+\.\d+)/i);
      if (!m) {
        return true;
      }
      const op = m[1] || '=';
      const target = parseVersion(m[2]);
      if (!target) {
        return true;
      }
      if (op === '^') {
        const upper: Version = [target[0] + 1, 0, 0];
        return (
          compareVersions(host, target) >= 0 && compareVersions(host, upper) < 0
        );
      }
      if (op === '~') {
        const upper: Version = [target[0], target[1] + 1, 0];
        return (
          compareVersions(host, target) >= 0 && compareVersions(host, upper) < 0
        );
      }
      const c0 = compareVersions(host, target);
      if (op === '>=') return c0 >= 0;
      if (op === '<=') return c0 <= 0;
      if (op === '>') return c0 > 0;
      if (op === '<') return c0 < 0;
      return c0 === 0;
    });
  };

  return range.split('||').some((clause) => evalClause(clause));
}

/**
 * API-compatibility gate. Checks the host *application* version against the
 * plugin's `spec.compatibleHostVersions`. Returns true when the range is
 * empty/unparseable, or on a local build (RELEASE_VERSION defaults to
 * v0.0.0-<sha>) or the unrendered `vite dev` placeholder, so development is
 * never gated. This axis is unrelated to the UI runtime — see
 * `satisfiesUiContract`.
 */
export function satisfiesHostVersion(version: string, range?: string): boolean {
  if (!range) {
    return true;
  }
  const host = parseVersion(version);
  if (!host || (host[0] === 0 && host[1] === 0 && host[2] === 0)) {
    return true;
  }
  return satisfiesRange(host, range);
}

/**
 * UI-contract gate. Checks the host's React version (the shared React major is
 * the only runtime a bundled-MUI plugin shares with the host — see issue #2661)
 * against the plugin's `spec.compatibleUiContractVersions`. Kept separate from
 * the host application version so a UI-breaking React major bump is caught on
 * its own axis. Returns true when the range is empty/unparseable.
 */
export function satisfiesUiContract(
  reactVersion: string,
  range?: string
): boolean {
  if (!range) {
    return true;
  }
  const host = parseVersion(reactVersion);
  if (!host) {
    return true;
  }
  return satisfiesRange(host, range);
}
