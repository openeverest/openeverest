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
import axios, { AxiosError } from 'axios';
import { enqueueSnackbar } from 'notistack';

const BASE_URL = '/v1/';
const DEFAULT_ERROR_MESSAGE = 'Something went wrong';
const MISSING_MALFORMED_JWT_MESSAGE = 'missing or malformed jwt';
const MAX_ERROR_MESSAGE_LENGTH = 120;
let errorInterceptor: number | null = null;
let authInterceptor: number | null = null;

export const api = axios.create({
  baseURL: BASE_URL,
});

export const addApiErrorInterceptor = () => {
  if (errorInterceptor === null) {
    errorInterceptor = api.interceptors.response.use(
      (response) => response,
      (error: AxiosError<{ message?: string }>) => {
        if (
          error.response &&
          error.response.status >= 400 &&
          error.response.status <= 500
        ) {
          let message = error.response.data?.message ?? DEFAULT_ERROR_MESSAGE;
          let notificationsDisabled =
            error.config?.disableNotifications ?? error.status === 429;

          if (typeof notificationsDisabled === 'function') {
            notificationsDisabled = notificationsDisabled(error);
          }

          if (
            error.response.status === 401 ||
            (error.response.status === 400 &&
              message.includes(MISSING_MALFORMED_JWT_MESSAGE))
          ) {
            location.href = '/logout';
            return;
          }

          if (!notificationsDisabled) {
            message = message.trim();
            if (message.length > MAX_ERROR_MESSAGE_LENGTH) {
              message = `${message.substring(0, MAX_ERROR_MESSAGE_LENGTH)}...`;
            }

            enqueueSnackbar(message, {
              variant: 'error',
            });
          }
        }

        return Promise.reject(error);
      }
    );
  }
};

export const removeApiErrorInterceptor = () => {
  if (errorInterceptor !== null) {
    api.interceptors.response.eject(errorInterceptor);
    errorInterceptor = null;
  }
};

export const addApiAuthInterceptor = () => {
  if (authInterceptor === null) {
    authInterceptor = api.interceptors.request.use((config) => {
      const token = localStorage.getItem('everestToken');
      if (token) {
        config.headers['Authorization'] = `Bearer ${token}`;
      }

      return config;
    });
  }
};

export const removeApiAuthInterceptor = () => {
  if (authInterceptor !== null) {
    api.interceptors.response.eject(authInterceptor);
    authInterceptor = null;
  }
};
