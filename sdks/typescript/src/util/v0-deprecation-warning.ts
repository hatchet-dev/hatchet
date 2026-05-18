/**
 * v0 SDK root-import deprecation warnings.
 *
 * The legacy `workflow` and `step` re-exports under the root specifier need
 * to keep nagging consumers to migrate to v1, but the original implementation
 * used `console.warn` at module evaluation time, which:
 *   - cannot be silenced by Node's standard `--no-deprecation`,
 *     `--no-warnings`, or `--no-warnings=DeprecationWarning` flags;
 *   - has no stable `code` for `process.on('warning', ...)` handlers; and
 *   - is duplicated when both submodules are loaded (which happens for
 *     anyone importing from the root, since `index.ts` re-exports both).
 *
 * Switching to `process.emitWarning` with a fixed code makes the warnings
 * filterable, dedupable, and consistent with the rest of Node's deprecation
 * surface, while still being visible by default.
 */

export const V0_DEPRECATION_CODE = 'HATCHET_V0_REMOVED';
const MIGRATION_URL = 'https://docs.hatchet.run/home/v1-sdk-improvements';

/** Tracks submodules whose warning has already been emitted this process. */
const emittedSubmodules = new Set<string>();

/** Reset hook for tests — not part of the public API. */
export function _resetEmittedV0Warnings(): void {
  emittedSubmodules.clear();
}

/**
 * Emit a deduplicated v0-removal deprecation warning for a given submodule.
 *
 * Each unique `submodule` is emitted at most once per process. Uses
 * `process.emitWarning` when available so consumers can suppress or
 * filter via standard Node mechanisms.
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

  if (typeof process !== 'undefined' && typeof process.emitWarning === 'function') {
    process.emitWarning(message, {
      type: 'DeprecationWarning',
      code: V0_DEPRECATION_CODE,
      detail,
    });
    return;
  }

  // Fallback for runtimes without `process.emitWarning`.
  console.warn(`[${V0_DEPRECATION_CODE}] ${message}${detail ? `\n${detail}` : ''}`);
}
