// C0 controls other than tab and newline, and DEL: backspace and ANSI escapes among them
// eslint-disable-next-line no-control-regex -- matching control characters is the point
const CONTROL = /[\u0000-\u0008\u000b-\u001f\u007f]/g;

/**
 * Shows the control characters in text the endpoint controls as visible escapes (`\x1b`), so
 * a viewer that interprets them, like the log viewer's ANSI rendering, cannot be made to erase
 * or restyle the text around them.
 */
export function escapeControls(text: string): string {
  return text.replace(
    CONTROL,
    (c) => `\\x${c.charCodeAt(0).toString(16).padStart(2, '0')}`,
  );
}
