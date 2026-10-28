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
