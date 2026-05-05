import { coerce } from 'semver';

export const isVersion84x = (version?: string): boolean => {
  const s = coerce(version);
  return !!s && s.major === 8 && s.minor === 4;
};
