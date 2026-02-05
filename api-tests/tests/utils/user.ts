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

import {
  API_CI_TOKEN,
  API_TEST_TOKEN,
  CI_USER,
  CI_PASSWORD,
  TEST_USER,
  TEST_PASSWORD,
} from '@root/constants';
import {APIRequestContext, expect} from "@playwright/test";

// Login functions
export const login = async (
  request: APIRequestContext,
  user: string,
  password: string,
  env_name: string
) => {
  const resp = await request.post(`/v1/session`, {data: {username: user, password: password}});
  expect(resp.ok()).toBeTruthy();
  process.env[env_name] = (await resp.json())['token']
};

export const loginCIUser = async (request: APIRequestContext) => {
  await login(request, CI_USER, CI_PASSWORD, API_CI_TOKEN);
};

export const loginTESTUser = async (request: APIRequestContext) => {
  await login(request, TEST_USER, TEST_PASSWORD, API_TEST_TOKEN);
};

// Logout functions
const logout = async (
  request: APIRequestContext,
  env_name: string
) => {
  const resp = await request.delete(`/v1/session`);
  expect(resp.ok()).toBeTruthy();
  delete process.env[env_name];
};

export const logoutCIUser = async (request: APIRequestContext) => {
  await logout(request, API_CI_TOKEN);
};
