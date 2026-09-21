"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import {
  CONSENT_COOKIE,
  CONSENT_COOKIE_MAX_AGE,
  REGION_COOKIE,
  defaultStatusForRegion,
  readCookie,
  resolveRegion,
  writeCookie,
  type ConsentRegion,
  type ConsentStatus,
} from "@/lib/consent";

export const CONSENT_CHANGE_EVENT = "cookie-consent-change";

interface ConsentContextType {
  /**
   * The consent we are acting on: the visitor's explicit choice when they have
   * made one, otherwise the default for their region.
   */
  consentStatus: ConsentStatus;
  region: ConsentRegion;
  /** False while the visitor is running on the region default. */
  hasExplicitChoice: boolean;
  isHydrated: boolean;
  hasConsent: boolean;
  grantConsent: () => void;
  denyConsent: () => void;
}

/**
 * Superseded by the cross-subdomain `ht_consent` cookie. Read once on mount so
 * returning visitors keep the decision they already made, then cleared. The
 * banner used to write all three of these; `ht_consent` is now the only store.
 */
const LEGACY_LOCAL_STORAGE_KEY = "cookie_consent";
const LEGACY_COOKIE_NAME = "cookieConsent";

const ConsentContext = createContext<ConsentContextType | undefined>(undefined);

function readStoredChoice(): ConsentStatus | null {
  const fromCookie = readCookie(CONSENT_COOKIE);
  if (fromCookie === "granted" || fromCookie === "denied") return fromCookie;

  try {
    const legacy = localStorage.getItem(LEGACY_LOCAL_STORAGE_KEY);
    if (legacy === "yes") return "granted";
    if (legacy === "no") return "denied";
  } catch {
    // Storage can be unavailable (Safari private mode); fall through.
  }

  // The old banner set `cookieConsent=true` only on accept.
  if (readCookie(LEGACY_COOKIE_NAME) === "true") return "granted";

  return null;
}

function clearLegacyStores() {
  try {
    localStorage.removeItem(LEGACY_LOCAL_STORAGE_KEY);
  } catch {
    // noop
  }
  // eslint-disable-next-line no-restricted-syntax
  document.cookie = `${LEGACY_COOKIE_NAME}=; path=/; max-age=0`;
}

export function ConsentProvider({ children }: { children: ReactNode }) {
  const [explicitStatus, setExplicitStatus] = useState<ConsentStatus | null>(null);
  const [region, setRegion] = useState<ConsentRegion>("restricted");
  const [isHydrated, setIsHydrated] = useState(false);

  const sync = useCallback(() => {
    setRegion(resolveRegion(readCookie(REGION_COOKIE)));
    setExplicitStatus(readStoredChoice());
  }, []);

  useEffect(() => {
    const stored = readStoredChoice();
    if (stored) {
      // Re-persist so a legacy value lands in the shared cookie, then drop the
      // origin-scoped copies.
      writeCookie(CONSENT_COOKIE, stored, { maxAgeSeconds: CONSENT_COOKIE_MAX_AGE });
    }
    clearLegacyStores();
    sync();
    setIsHydrated(true);

    const onStorage = (e: StorageEvent) => {
      if (e.key === LEGACY_LOCAL_STORAGE_KEY) sync();
    };
    window.addEventListener("storage", onStorage);
    window.addEventListener(CONSENT_CHANGE_EVENT, sync);

    return () => {
      window.removeEventListener("storage", onStorage);
      window.removeEventListener(CONSENT_CHANGE_EVENT, sync);
    };
  }, [sync]);

  const updateConsent = useCallback((next: ConsentStatus) => {
    writeCookie(CONSENT_COOKIE, next, { maxAgeSeconds: CONSENT_COOKIE_MAX_AGE });
    setExplicitStatus(next);
    window.dispatchEvent(new Event(CONSENT_CHANGE_EVENT));
  }, []);

  const value = useMemo<ConsentContextType>(() => {
    const consentStatus = explicitStatus ?? defaultStatusForRegion(region);

    return {
      consentStatus,
      region,
      hasExplicitChoice: explicitStatus !== null,
      isHydrated,
      hasConsent: consentStatus === "granted",
      grantConsent: () => updateConsent("granted"),
      denyConsent: () => updateConsent("denied"),
    };
  }, [explicitStatus, region, isHydrated, updateConsent]);

  return <ConsentContext.Provider value={value}>{children}</ConsentContext.Provider>;
}

export function useConsent() {
  const context = useContext(ConsentContext);
  if (context === undefined) {
    throw new Error("useConsent must be used within a ConsentProvider");
  }
  return context;
}
