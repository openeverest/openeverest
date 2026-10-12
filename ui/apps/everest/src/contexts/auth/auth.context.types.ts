import { JwtPayload } from 'jwt-decode';

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
  setRedirectRoute: (route: string) => void;
  authStatus: UserAuthStatus;
  redirectRoute: string | null;
  isSsoEnabled: boolean;
}

export interface AuthProviderProps {
  children: React.ReactNode;
  isSsoEnabled: boolean;
}

export type EverestJwtPayload = JwtPayload & { oidc_issuer?: string };
