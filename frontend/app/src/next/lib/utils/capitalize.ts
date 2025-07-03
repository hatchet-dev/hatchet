export function capitalize(s: string) {
  if (!s) {
    return '';
  } else if (s.length === 0) {
    return s;
  } else if (s.length === 1) {
    return s.charAt(0).toUpperCase();
  }

  return s.charAt(0).toUpperCase() + s.substring(1).toLowerCase();
}
