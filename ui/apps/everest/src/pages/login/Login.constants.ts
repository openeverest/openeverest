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

import { z } from 'zod';
import { Messages } from './Login.messages';

export enum LoginFields {
  username = 'username',
  password = 'password',
}

export const loginSchema = z.object({
  [LoginFields.username]: z.string().min(1, { message: 'Required' }),
  [LoginFields.password]: z.string().min(1, { message: 'Required' }),
});

export type LoginFormType = z.infer<typeof loginSchema>;

export const LOGIN_LINKS = [
  {
    label: Messages.links.community,
    href: 'https://openeverest.io/#community',
  },
  {
    label: Messages.links.github,
    href: 'https://github.com/openeverest/openeverest',
  },
  {
    label: Messages.links.documentation,
    href: 'https://openeverest.io/docs',
  },
  { label: Messages.links.support, href: 'https://openeverest.io/support' },
];
