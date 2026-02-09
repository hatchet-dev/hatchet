"use client";

import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";

type ConsentStatus = "yes" | "no" | "undecided";

interface ConsentContextType {
  consentStatus: ConsentStatus;
  hasConsent: boolean;
}

const ConsentContext = createContext<ConsentContextType | undefined>(undefined);

export function ConsentProvider({ children }: { children: ReactNode }) {
  const [consentStatus, setConsentStatus] =
    useState<ConsentStatus>("undecided");

  useEffect(() => {
    const checkConsent = () => {
      const consent = localStorage.getItem("cookie_consent");
      if (consent === "yes" || consent === "no") {
        setConsentStatus(consent as ConsentStatus);
      } else {
        setConsentStatus("undecided");
      }
    };

    checkConsent();

    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === "cookie_consent") {
        checkConsent();
      }
    };

    const handleConsentChange = () => {
      checkConsent();
    };

    window.addEventListener("storage", handleStorageChange);
    window.addEventListener("cookie-consent-change", handleConsentChange);

    return () => {
      window.removeEventListener("storage", handleStorageChange);
      window.removeEventListener("cookie-consent-change", handleConsentChange);
    };
  }, []);

  const hasConsent = consentStatus === "yes";

  return (
    <ConsentContext.Provider value={{ consentStatus, hasConsent }}>
      {children}
    </ConsentContext.Provider>
  );
}

export function useConsent() {
  const context = useContext(ConsentContext);
  if (context === undefined) {
    throw new Error("useConsent must be used within a ConsentProvider");
  }
  return context;
}
