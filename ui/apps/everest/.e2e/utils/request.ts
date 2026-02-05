// everest
// Copyright (C) 2023 Percona LLC
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

import { APIResponse } from '@playwright/test';

export const doBackupCall = async (
  fn: () => Promise<APIResponse>,
  retries = 3
) => {
  if (retries === 0) {
    return Promise.reject();
  }

  try {
    const response = await fn();
    const statusText = await response.json();
    const ok = response.ok();

    if (ok) {
      return Promise.resolve();
    }

    if (
      response.status() === 409 ||
      (response.status() === 400 &&
        statusText?.message &&
        statusText?.message.includes('same url'))
    ) {
      return Promise.resolve();
    }

    if (response.status() !== 201 && response.status() !== 200) {
      return Promise.reject();
    }

    if (statusText && statusText.message) {
      if (statusText.message.includes('Could not read')) {
        if (retries > 0) {
          return doBackupCall(fn, retries - 1);
        }
      } else {
        return Promise.resolve();
      }
    }
    return Promise.reject();
  } catch (error) {
    return Promise.reject();
  }
};
