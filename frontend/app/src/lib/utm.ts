export const UTM_PARAMS_KEY = 'utm_params';

const UTM_KEYS = [
  'utm_source',
  'utm_medium',
  'utm_campaign',
  'utm_term',
  'utm_content',
];

const MAX_LENGTH = 128;
const ALLOWED_PATTERN = /^[a-zA-Z0-9 ._-]+$/;

export function captureUtmParams(search: string) {
  const params = new URLSearchParams(search);
  const utms: Record<string, string> = {};
  for (const key of UTM_KEYS) {
    const value = params.get(key)?.trim();
    if (value && value.length <= MAX_LENGTH && ALLOWED_PATTERN.test(value)) {
      utms[key] = value;
    }
  }
  if (Object.keys(utms).length > 0) {
    localStorage.setItem(UTM_PARAMS_KEY, JSON.stringify(utms));
  }
}

export function readUtmParams(): Record<string, string> | null {
  const raw = localStorage.getItem(UTM_PARAMS_KEY);
  if (!raw) {
    return null;
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== 'object' || parsed === null) {
      return null;
    }
    const utms: Record<string, string> = {};
    for (const key of UTM_KEYS) {
      const value = (parsed as Record<string, unknown>)[key];
      if (typeof value === 'string') {
        utms[key] = value;
      }
    }
    return Object.keys(utms).length > 0 ? utms : null;
  } catch {
    return null;
  }
}

export function clearUtmParams() {
  localStorage.removeItem(UTM_PARAMS_KEY);
}
