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

// Playwright fetcher for orval-generated client.
// Bridges orval's fetch-style calls to Playwright's APIRequestContext.
import type { APIRequestContext, APIResponse } from '@playwright/test'

const API_PREFIX = '/v1'

let _request: APIRequestContext

export const setRequestContext = (request: APIRequestContext) => {
  _request = request
}

export const getRequestContext = (): APIRequestContext => {
  if (!_request) {
    throw new Error('Request context not set. Call setRequestContext() first.')
  }
  return _request
}

type HttpMethod = 'get' | 'post' | 'put' | 'patch' | 'delete'

/**
 * Fetcher for orval-generated client.
 * Translates fetch-style calls to Playwright's APIRequestContext
 * and returns the raw Playwright APIResponse.
 */
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export const playwrightFetcher = async <T>(
  url: string,
  options: {
    method?: string
    headers?: Record<string, string>
    body?: string
    signal?: AbortSignal
  } = {},
): Promise<APIResponse> => {
  const request = getRequestContext()
  const method = (options.method?.toLowerCase() ?? 'get') as HttpMethod
  const fullUrl = `${API_PREFIX}${url}`

  const fetchOptions: Record<string, unknown> = {}
  if (options.body) {
    fetchOptions.data = JSON.parse(options.body)
  }
  if (options.headers) {
    fetchOptions.headers = options.headers
  }

  return await request[method](fullUrl, fetchOptions)
}

export default playwrightFetcher
