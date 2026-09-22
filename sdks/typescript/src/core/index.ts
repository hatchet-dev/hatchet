/**
 * The core entry point: the application client for runtimes without gRPC, built on the
 * Connect transport seam. The entry resolves today so that `@hatchet-dev/typescript-sdk/core`
 * is a stable import path; the client it exports is added on top.
 *
 * @module Core
 */

/** Bumped whenever the surface exported from this entry changes incompatibly. */
export const CORE_ENTRY_VERSION = 0;
