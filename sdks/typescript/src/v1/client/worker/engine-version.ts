import { semverLessThan } from './deprecated/deprecation';

export const MinEngineVersion = {
  SLOT_CONFIG: 'v0.78.23',
  DURABLE_EVICTION: 'v0.80.0',
  RETRY_AFTER: 'v0.111.0',
} as const;

export function supportsRetryAfter(engineVersion: string | undefined): boolean {
  if (!engineVersion) return false;
  return !semverLessThan(engineVersion, MinEngineVersion.RETRY_AFTER);
}

export function supportsEviction(engineVersion: string | undefined): boolean {
  if (!engineVersion) return false;
  return !semverLessThan(engineVersion, MinEngineVersion.DURABLE_EVICTION);
}
