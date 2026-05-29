export const REFERRAL_CODE_KEY = 'referral_key';

const MAX_LENGTH = 64;
const ALLOWED_PATTERN = /^[a-zA-Z0-9_-]+$/;

export function sanitizeReferralCode(
  raw: string | null | undefined,
): string | null {
  if (!raw) {
    return null;
  }
  const trimmed = raw.trim();
  if (trimmed.length === 0 || trimmed.length > MAX_LENGTH) {
    return null;
  }
  if (!ALLOWED_PATTERN.test(trimmed)) {
    return null;
  }
  return trimmed;
}
