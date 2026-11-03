import {
  SupportGrantRole,
  SupportSession,
} from '@/lib/api/generated/control-plane/data-contracts';
import { useCallback, useSyncExternalStore } from 'react';

// Per-tab (sessionStorage) so a support session never leaks into the staff
// member's own tabs, but survives a reload of the tab it was redeemed in.
export const SUPPORT_SESSION_STORAGE_KEY = 'hatchet-support-session';
// Kept after the session is cleared so the "ended" page can still link back
// to the console the session was started from.
const LAST_CONSOLE_URL_STORAGE_KEY = 'hatchet-support-last-console-url';

export const SUPPORT_SESSION_QUERY_KEY_PREFIX = 'support-session';

export const SUPPORT_SESSION_RENEW_BEFORE_MS = 3 * 60_000;

type StoredSupportSession = {
  session: SupportSession;
  // Set once the token was rejected, so renewal is not retried.
  expired?: boolean;
};

type Listener = () => void;

const listeners = new Set<Listener>();

// Parsed view of the storage value, keyed by the raw string so repeated reads
// return the same reference (useSyncExternalStore needs a stable snapshot).
let lastRaw: string | null = null;
let lastParsed: StoredSupportSession | null = null;

function readStorage(): string | null {
  try {
    return sessionStorage.getItem(SUPPORT_SESSION_STORAGE_KEY);
  } catch {
    return null;
  }
}

function writeStorage(value: StoredSupportSession | null) {
  try {
    if (value) {
      sessionStorage.setItem(
        SUPPORT_SESSION_STORAGE_KEY,
        JSON.stringify(value),
      );
    } else {
      sessionStorage.removeItem(SUPPORT_SESSION_STORAGE_KEY);
    }
  } catch {
    /* empty */
  }
}

function parseStored(raw: string): StoredSupportSession | null {
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object') {
      return null;
    }
    const stored = parsed as Partial<StoredSupportSession>;
    if (
      !stored.session ||
      typeof stored.session.token !== 'string' ||
      typeof stored.session.apiUrl !== 'string' ||
      typeof stored.session.expiresAt !== 'string' ||
      !stored.session.grant
    ) {
      return null;
    }
    return { session: stored.session, expired: stored.expired === true };
  } catch {
    return null;
  }
}

function getStored(): StoredSupportSession | null {
  const raw = readStorage();
  if (raw !== lastRaw) {
    lastRaw = raw;
    lastParsed = raw ? parseStored(raw) : null;
  }
  return lastParsed;
}

function notify() {
  listeners.forEach((listener) => listener());
}

function subscribe(listener: Listener) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function getSupportSession(): SupportSession | null {
  return getStored()?.session ?? null;
}

export function setSupportSession(session: SupportSession) {
  writeStorage({ session });
  try {
    if (session.consoleUrl) {
      sessionStorage.setItem(LAST_CONSOLE_URL_STORAGE_KEY, session.consoleUrl);
    }
  } catch {
    /* empty */
  }
  notify();
}

export function clearSupportSession() {
  writeStorage(null);
  notify();
}

export function markSupportSessionExpired() {
  const stored = getStored();
  if (!stored || stored.expired) {
    return;
  }
  writeStorage({ ...stored, expired: true });
  notify();
}

export function isSupportSessionExpired(
  session: SupportSession | null = getSupportSession(),
  now: number = Date.now(),
): boolean {
  if (!session) {
    return false;
  }
  if (getStored()?.expired) {
    return true;
  }
  const expiry = new Date(session.expiresAt).getTime();
  return Number.isNaN(expiry) || expiry <= now;
}

export function getLastSupportConsoleUrl(): string | null {
  try {
    return sessionStorage.getItem(LAST_CONSOLE_URL_STORAGE_KEY);
  } catch {
    return null;
  }
}

export function supportRoleLabel(role: SupportGrantRole): string {
  return role === SupportGrantRole.SUPPORT_OPERATOR ? 'operator' : 'viewer';
}

export function useSupportSession() {
  const stored = useSyncExternalStore(subscribe, getStored, getStored);
  const session = stored?.session ?? null;

  const setSession = useCallback(
    (next: SupportSession) => setSupportSession(next),
    [],
  );
  const clear = useCallback(() => clearSupportSession(), []);

  return {
    session,
    isSupport: session !== null,
    isExpired: isSupportSessionExpired(session),
    setSession,
    clear,
  };
}
