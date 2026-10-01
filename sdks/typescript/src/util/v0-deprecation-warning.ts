/**
 * The v0 SDK deprecation warning, emitted when a v0 workflow is actually used (registered on a
 * worker or put to the engine), never when a module is merely imported: the root entry still
 * re-exports the v0 `workflow` and `step` modules for consumers that only use v1 APIs, and
 * loading them must stay silent.
 *
 * `process.emitWarning` with a fixed code keeps the warning filterable with Node's standard
 * `--no-deprecation` and `--no-warnings` flags and addressable from `process.on('warning')`.
 */

export const V0_DEPRECATION_CODE = 'HATCHET_V0_REMOVED';
const MIGRATION_URL = 'https://docs.hatchet.run/home/v1-sdk-improvements';

const emittedSubmodules = new Set<string>();

/** Reset hook for tests. Not part of the public API. */
export function _resetEmittedV0Warnings(): void {
  emittedSubmodules.clear();
}

function fallbackConsoleWarn(message: string, detail?: string): void {
  console.warn(`[${V0_DEPRECATION_CODE}] ${message}${detail ? `\n${detail}` : ''}`);
}

/**
 * Emit a deduplicated v0-removal deprecation warning for a given submodule.
 *
 * Each unique `submodule` is emitted at most once per process. Uses
 * `process.emitWarning` when available so consumers can suppress or filter
 * via standard Node mechanisms.
 *
 * Importing the SDK must never abort module evaluation, since the root
 * specifier still re-exports v0 `workflow` and `step` for transitive
 * consumers who only use v1 APIs. Two cases would otherwise crash them:
 *
 *   1. `process.throwDeprecation` (set by `--throw-deprecation` or directly).
 *      Node queues a `throw warning` on the next tick after `emitWarning`
 *      returns, so a `try`/`catch` around the call would not catch it. We
 *      check the flag up front and route to `console.warn` instead.
 *   2. Runtimes that don't expose `process.emitWarning` at all (older
 *      browsers, certain bundler shims). Same fallback applies.
 *
 * The remaining `try`/`catch` is belt-and-suspenders for non-Node hosts
 * where a polyfilled `emitWarning` could throw synchronously.
 *
 * @param submodule - The legacy v0 submodule being imported (e.g. "workflow", "step").
 * @param detail    - Optional follow-up sentence appended to the warning detail.
 */
export function emitV0RemovedWarning(submodule: string, detail?: string): void {
  if (emittedSubmodules.has(submodule)) {
    return;
  }
  emittedSubmodules.add(submodule);

  const message =
    `The v0 SDK, including the ${submodule} module, has been deprecated and was removed in v1.14.0. ` +
    `Please migrate to the v1 SDK: ${MIGRATION_URL}`;

  const hasProcess = typeof process !== 'undefined';
  const hasEmitWarning = hasProcess && typeof process.emitWarning === 'function';
  const willThrowAsync = hasProcess && process.throwDeprecation === true;

  if (!hasEmitWarning || willThrowAsync) {
    fallbackConsoleWarn(message, detail);
    return;
  }

  try {
    process.emitWarning(message, {
      type: 'DeprecationWarning',
      code: V0_DEPRECATION_CODE,
      detail,
    });
  } catch {
    fallbackConsoleWarn(message, detail);
  }
}
