import { fileURLToPath } from 'url';
import path from 'path';

const {
  EVEREST_BUCKETS_NAMESPACES_MAP,
  EVEREST_DIR,
  TAG_FOR_UPGRADE,
  FB_BUILD,
} = process.env;

type BucketsNamespaceMap = [string, string][];

export const CI_USER_STORAGE_STATE_FILE = path.join(
  path.dirname(fileURLToPath(import.meta.url)),
  '.auth/',
  'ci_user.json'
);
export const SESSION_USER_STORAGE_STATE_FILE = path.join(
  path.dirname(fileURLToPath(import.meta.url)),
  '.auth/',
  'session_user.json'
);

export enum EVEREST_CI_NAMESPACES {
  EVEREST_UI = 'everest',
  PSMDB_ONLY = 'psmdb-only',
  PXC_ONLY = 'pxc-only',
  PG_ONLY = 'pg-only',
}

export const technologyMap: Record<string, string> = {
  psmdb: 'MongoDB',
  pxc: 'MySQL',
  postgresql: 'PostgreSQL',
};

export const getBucketNamespacesMap = (): BucketsNamespaceMap =>
  JSON.parse(EVEREST_BUCKETS_NAMESPACES_MAP);

export const everestdir = EVEREST_DIR;
export const everestTagForUpgrade = TAG_FOR_UPGRADE;
export const everestFeatureBuildForUpgrade = FB_BUILD;

const second = 1_000;
const minute = 60 * second;

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
