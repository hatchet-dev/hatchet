/**
 * Generic time-aware deprecation helper.
 *
 * Timeline (from a given start date, with configurable windows):
 *   0 to warnDays:              WARNING logged once per feature
 *   warnDays to errorDays:      ERROR logged once per feature
 *   after errorDays:            throws an error 1-in-5 calls (20% chance)
 *
 * Defaults: warnDays=90, errorDays=undefined (error phase disabled unless set).
 */

import { Logger } from '@hatchet/util/logger';

const DEFAULT_WARN_DAYS = 90;
const MS_PER_DAY = 24 * 60 * 60 * 1000;

/** Tracks which features have already been logged (keyed by feature name). */
const alreadyLogged = new Set<string>();

export class DeprecationError extends Error {
  feature: string;

  constructor(feature: string, message: string) {
    super(`${feature}: ${message}`);
    this.name = 'DeprecationError';
    this.feature = feature;
  }
}

export interface DeprecationOpts {
  /** Days after start during which a warning is logged. Defaults to 90. */
  warnDays?: number;
  /** Days after start during which an error is logged.
   *  After this window, calls have a 20% chance of throwing.
   *  If undefined (default), the error/raise phase is never reached —
   *  the notice stays at error-level logging indefinitely. */
  errorDays?: number;
}

/**
 * Emit a time-aware deprecation notice.
 *
 * @param feature - A short identifier for deduplication (each feature logs once).
 * @param message - The human-readable deprecation message.
 * @param start   - The Date when the deprecation window began.
 * @param logger  - A Logger instance for outputting warnings/errors.
 * @param opts    - Optional configuration for time windows.
 * @throws DeprecationError after the errorDays window (~20% chance).
 */
/**
 * Parses a semver string like "v0.78.23" into [major, minor, patch].
 * Returns [0, 0, 0] if parsing fails.
 */
export function parseSemver(v: string): [number, number, number] {
  let s = v.startsWith('v') ? v.slice(1) : v;
  const dashIdx = s.indexOf('-');
  if (dashIdx !== -1) s = s.slice(0, dashIdx);
  const parts = s.split('.');
  if (parts.length !== 3) return [0, 0, 0];
  return [parseInt(parts[0], 10) || 0, parseInt(parts[1], 10) || 0, parseInt(parts[2], 10) || 0];
}

/**
 * Returns true if semver string a is strictly less than b.
 */
export function semverLessThan(a: string, b: string): boolean {
  const [aMaj, aMin, aPat] = parseSemver(a);
  const [bMaj, bMin, bPat] = parseSemver(b);
  if (aMaj !== bMaj) return aMaj < bMaj;
  if (aMin !== bMin) return aMin < bMin;
  return aPat < bPat;
}

export function emitDeprecationNotice(
  feature: string,
  message: string,
  start: Date,
  logger: Logger,
  opts?: DeprecationOpts
): void {
  const warnMs = (opts?.warnDays ?? DEFAULT_WARN_DAYS) * MS_PER_DAY;
  const errorDays = opts?.errorDays;
  const errorMs = errorDays != null ? errorDays * MS_PER_DAY : undefined;
  const elapsed = Date.now() - start.getTime();

  if (elapsed < warnMs) {
    // Phase 1: warning
    if (!alreadyLogged.has(feature)) {
      logger.warn(message);
      alreadyLogged.add(feature);
    }
  } else if (errorMs === undefined || elapsed < errorMs) {
    // Phase 2: error-level log (indefinite when errorDays is not set)
    if (!alreadyLogged.has(feature)) {
      logger.error(`${message} This fallback will be removed soon. Upgrade immediately.`);
      alreadyLogged.add(feature);
    }
  } else {
    // Phase 3: throw 1-in-5 times
    if (!alreadyLogged.has(feature)) {
      logger.error(`${message} This fallback is no longer supported and will fail intermittently.`);
      alreadyLogged.add(feature);
    }

    if (Math.random() < 0.2) {
      throw new DeprecationError(feature, message);
    }
  }
}
