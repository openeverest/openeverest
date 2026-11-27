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

export type UserAuthStatus =
  | 'loggingIn'
  | 'loggingOut'
  | 'loggedIn'
  | 'loggedOut'
  | 'unknown';

export type AuthMode = 'manual' | 'sso';
export type ManualAuthArgs = { username: string; password: string };
export interface AuthContextProps {
  login: (mode: AuthMode, manualAuthArgs?: ManualAuthArgs) => void;
  logout: () => void;
  setRedirectRoute: (route: string | null) => void;
  authStatus: UserAuthStatus;
  redirectRoute: string | null;
  isSsoEnabled: boolean;
  isLogoutIntentional?: boolean;
}

export interface AuthProviderProps {
  children: React.ReactNode;
  isSsoEnabled: boolean;
}
