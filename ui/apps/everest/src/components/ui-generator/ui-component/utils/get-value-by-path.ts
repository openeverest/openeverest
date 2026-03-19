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

const resolvePath = (path: string | string[]): string | undefined => {
  if (typeof path === 'string') {
    return path || undefined;
  }

  if (Array.isArray(path)) {
    return path.find((p): p is string => typeof p === 'string' && !!p);
  }

  return undefined;
};

export const getValueByPath = (
  obj: unknown,
  path: string | string[]
): unknown => {
  const resolvedPath = resolvePath(path);

  if (!resolvedPath) {
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.error('[getValueByPath] Invalid path argument', {
        path,
        pathType: typeof path,
      });
    }
    return undefined;
  }

  if (!obj || typeof obj !== 'object') {
    return undefined;
  }

  return resolvedPath
    .split('.')
    .reduce<unknown>(
      (acc, part) =>
        acc && typeof acc === 'object'
          ? (acc as Record<string, unknown>)[part]
          : undefined,
      obj
    );
};
