/**
 * Empty task output - satisfies JsonObject for concurrency workflows
 * that don't return meaningful data (used for rate-limiting tests).
 */
export type EmptyTaskOutput = Record<string, never>;
