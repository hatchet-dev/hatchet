/** Mock OCR - no external dependencies */

export function parseDocument(content: Uint8Array): string {
  return `Parsed text from ${content.length} bytes`;
}
