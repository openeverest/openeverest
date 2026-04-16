import 'axios';

declare module 'axios' {
  export interface AxiosRequestConfig {
    disableNotifications?: boolean | ((error: AxiosError) => boolean);
  }
}
