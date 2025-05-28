export function isValidUUID(uuid: string): boolean {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(uuid);
}

export function validateUUID(uuid: string, errorMessage = 'Invalid UUID'): string {
  if (!isValidUUID(uuid)) {
    throw new Error(errorMessage);
  }
  return uuid;
}
