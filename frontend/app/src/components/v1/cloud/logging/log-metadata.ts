export function hasLogMetadata(
  metadata: Record<string, unknown> | undefined,
): metadata is Record<string, unknown> {
  return !!metadata && Object.keys(metadata).length > 0;
}

export function formatLogMetadata(metadata: Record<string, unknown>): string {
  return JSON.stringify(metadata, null, 2);
}
