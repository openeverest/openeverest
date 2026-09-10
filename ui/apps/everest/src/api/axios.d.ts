import 'axios';

declare module 'axios' {
  export interface AxiosRequestConfig {
    disableNotifications?: boolean | ((error: AxiosError) => boolean);
    // Set once the 401 handler has already replayed this request after a token
    // refresh, so a second 401 can't loop.
    _retry?: boolean;
  }
}
