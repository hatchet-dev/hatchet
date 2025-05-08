export const SHEET_PARAM_KEY = 'sheet';

export function encodeSheetProps(props: unknown): string {
  return btoa(JSON.stringify(props));
}

export function decodeSheetProps(encoded: string): unknown {
  try {
    const decoded = atob(encoded);
    return JSON.parse(decoded);
  } catch (e) {
    console.error('Failed to decode sheet parameter:', e);
    return undefined;
  }
} 