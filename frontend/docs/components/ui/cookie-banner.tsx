"use client";

import { useCallback, useEffect, useState } from "react";
import { cn } from "@/lib/utils";
import { CookieIcon } from "@radix-ui/react-icons";
import posthog from "posthog-js";
import { useConsent } from "@/context/ConsentContext";

export default function CookieConsent() {
  const { region, isHydrated, hasExplicitChoice, grantConsent, denyConsent } =
    useConsent();
  const [isOpen, setIsOpen] = useState(false);
  const [hide, setHide] = useState(true);

  // In the EEA, the UK and Switzerland nothing non-essential has run yet, so
  // the banner asks. Everywhere else analytics is already on by default, so it
  // says so and offers the way out rather than requesting permission it has
  // already assumed.
  const isRequest = region === "restricted";

  useEffect(() => {
    if (!isHydrated) return;

    if (hasExplicitChoice) {
      setIsOpen(false);
      const timeout = setTimeout(() => setHide(true), 700);
      return () => clearTimeout(timeout);
    }

    setHide(false);
    setIsOpen(true);
  }, [isHydrated, hasExplicitChoice]);

  const dismiss = useCallback(() => {
    setIsOpen(false);
    setTimeout(() => setHide(true), 700);
  }, []);

  const acceptClick = useCallback(() => {
    posthog.capture("accept-cookies", { accepted: true });
    grantConsent();
    dismiss();
  }, [grantConsent, dismiss]);

  const declineClick = useCallback(() => {
    posthog.capture("accept-cookies", { accepted: false });
    denyConsent();
    dismiss();
  }, [denyConsent, dismiss]);

  const body = isRequest
    ? "We use cookies and similar technologies for analytics and marketing. You can allow these cookies or continue with only essential cookies."
    : "We use cookies and similar technologies for analytics and marketing. They are on by default here — you can opt out at any time.";
  const acceptLabel = isRequest ? "Accept" : "Got it";
  const declineLabel = isRequest ? "Decline" : "Opt out";

  return (
    <div
      className={cn(
        "fixed z-[200] bottom-0 left-0 right-0 sm:left-auto sm:right-4 sm:bottom-4 w-full sm:max-w-sm duration-700",
        !isOpen
          ? "transition-[opacity,transform] translate-y-8 opacity-0"
          : "transition-[opacity,transform] translate-y-0 opacity-100",
        hide && "hidden",
      )}
    >
      <div className="m-3 rounded-lg border border-fd-border bg-fd-popover text-fd-popover-foreground shadow-lg">
        <div className="p-4">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold">We use cookies</h2>
            <CookieIcon className="h-4 w-4 text-fd-muted-foreground" />
          </div>
          <p className="mt-2 text-[13px] leading-relaxed text-fd-muted-foreground">
            {body}{" "}
            <a
              href="https://hatchet.run/policies/cookie"
              className="underline underline-offset-2 hover:text-fd-foreground"
            >
              Learn more.
            </a>
          </p>
          <div className="mt-4 flex gap-2">
            <button
              type="button"
              onClick={acceptClick}
              className="h-8 flex-1 rounded-md bg-fd-primary text-[13px] font-medium text-white transition-colors hover:bg-fd-primary/90 dark:bg-white dark:text-fd-primary-foreground dark:hover:bg-white/90"
            >
              {acceptLabel}
            </button>
            <button
              type="button"
              onClick={declineClick}
              className="h-8 flex-1 rounded-md border border-fd-border text-[13px] font-medium text-fd-foreground transition-colors hover:bg-fd-accent"
            >
              {declineLabel}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
