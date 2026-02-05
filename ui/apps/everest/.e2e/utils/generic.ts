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

import { APIResponse, expect, Page } from '@playwright/test';
import { execSync } from 'child_process';
import { TIMEOUTS } from '@e2e/constants';

const { SELECT_DB, SELECT_SIZE } = process.env;

export const testPrefix = () => {
  let result = '';
  while (result.length < 16) {
    result += Math.random().toString(36).substring(2);
  }
  return result.substring(0, 16);
};

export const limitedSuffixedName = (name: string) => {
  return `${name}-${testPrefix()}`.substring(0, 21);
};

export const checkError = async (response: APIResponse) => {
  if (!response.ok() && response.status() !== 404) {
    console.log(`${response.url()}: `, await response.json());
  }
  // expect(response.status()).toBe(200);
  expect(response.ok()).toBeTruthy();
};

export const getDbOperatorVersionK8s = async (
  namespace: string,
  operator: string
) => {
  try {
    const command = `kubectl get deployment --namespace ${namespace} ${operator} -o jsonpath="{.spec.template.spec.containers[0].image}"`;
    const output = execSync(command).toString();
    const version = output.split(':')[1];
    return version;
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const shouldExecuteDBCombination = (db: string, size: number) => {
  return (
    (SELECT_DB ? SELECT_DB === db : true) &&
    (SELECT_SIZE ? SELECT_SIZE === size.toString() : true)
  );
};

export const goToUrl = async (page: Page, url: string) => {
  await page.goto(url);
  await page.waitForLoadState('load', { timeout: TIMEOUTS.ThirtySeconds });
};
