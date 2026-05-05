import {fileURLToPath} from 'url';
import path from 'path';

export const EVEREST_CI_NAMESPACE = 'everest',
  {
    CI_USER,
    CI_PASSWORD,
    TEST_USER,
    TEST_PASSWORD,
  } = process.env,
  CI_USER_STORAGE_STATE_FILE = path.join(
    path.dirname(fileURLToPath(import.meta.url)),
    '.auth',
    'ci_user.json'
  ),
  API_CI_TOKEN = 'API_CI_TOKEN',
  API_TEST_TOKEN = 'API_TEST_TOKEN',
  MONITORING_CONFIG_1 = 'pmm-conf-1',
  MONITORING_CONFIG_2 = 'pmm-conf-2';

const second = 1_000,
  minute = 60 * second;

export enum TIMEOUTS {
  FiveSeconds = 5 * second,
  TenSeconds = 10 * second,
  FifteenSeconds = 15 * second,
  ThirtySeconds = 30 * second,
  OneMinute = minute,
  ThreeMinutes = 3 * minute,
  FiveMinutes = 5 * minute,
  TenMinutes = 10 * minute,
  FifteenMinutes = 15 * minute,
  TwentyMinutes = 20 * minute,
  ThirtyMinutes = 30 * minute,
  SixtyMinutes = 60 * minute,
}
