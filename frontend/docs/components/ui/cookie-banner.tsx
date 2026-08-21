"use client";

import { useCallback, useEffect, useState } from "react";
import { cn } from "@/lib/utils";
import { Button } from "./button";
import { CookieIcon } from "@radix-ui/react-icons";
import posthog from "posthog-js";
import { useConsent } from "@/context/ConsentContext";

export default function CookieConsent({
  variant = "default",
  demo = false,
  onAcceptCallback = () => {},
  onDeclineCallback = () => {},
}) {
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

    if (hasExplicitChoice && !demo) {
      setIsOpen(false);
      const timeout = setTimeout(() => setHide(true), 700);
      return () => clearTimeout(timeout);
    }

    setHide(false);
    setIsOpen(true);
  }, [isHydrated, hasExplicitChoice, demo]);

  const dismiss = useCallback(() => {
    setIsOpen(false);
    setTimeout(() => setHide(true), 700);
  }, []);

  const acceptClick = useCallback(() => {
    posthog.capture("accept-cookies", { accepted: true });
    grantConsent();
    dismiss();
    onAcceptCallback();
  }, [grantConsent, dismiss, onAcceptCallback]);

  const declineClick = useCallback(() => {
    posthog.capture("accept-cookies", { accepted: false });
    denyConsent();
    dismiss();
    onDeclineCallback();
  }, [denyConsent, dismiss, onDeclineCallback]);

  const body = isRequest
    ? "We use cookies and similar technologies for analytics and marketing. You can allow these cookies or continue with only essential cookies."
    : "We use cookies and similar technologies for analytics and marketing. They are on by default here — you can opt out at any time.";
  const acceptLabel = isRequest ? "Accept" : "Got it";
  const declineLabel = isRequest ? "Decline" : "Opt out";

  // Default banner
  if (variant === "default") {
    return (
      <div
        className={cn(
          "fixed z-[200] bottom-0 left-0 right-0 sm:left-4 sm:bottom-4 w-full sm:max-w-md duration-700",
          !isOpen
            ? "transition-[opacity,transform] translate-y-8 opacity-0"
            : "transition-[opacity,transform] translate-y-0 opacity-100",
          hide && "hidden",
        )}
      >
        <div className="dark:bg-card bg-background rounded-md m-3 border border-border shadow-lg">
          <div className="grid gap-2">
            <div className="border-b border-border h-14 flex items-center justify-between p-4">
              <h1 className="text-lg font-medium">We use cookies</h1>
              <CookieIcon className="h-[1.2rem] w-[1.2rem]" />
            </div>
            <div className="p-4">
              <p className="text-sm font-normal text-start">
                {body}
                <br />
                <br />
                <a
                  href="https://hatchet.run/policies/cookie"
                  className="text-xs underline"
                >
                  Learn more.
                </a>
              </p>
            </div>
            <div className="flex gap-2 p-4 py-5 border-t border-border dark:bg-background/20">
              <Button onClick={acceptClick} className="w-full">
                {acceptLabel}
              </Button>
              <Button
                onClick={declineClick}
                className="w-full"
                variant="secondary"
              >
                {declineLabel}
              </Button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Small banner variant
  return (
    <div
      className={cn(
        "fixed z-[200] bottom-0 left-0 right-0 sm:left-4 sm:bottom-4 w-full sm:max-w-md duration-700",
        !isOpen
          ? "transition-[opacity,transform] translate-y-8 opacity-0"
          : "transition-[opacity,transform] translate-y-0 opacity-100",
        hide && "hidden",
      )}
    >
      <div className="m-3 dark:bg-card bg-background border border-border rounded-lg">
        <div className="flex items-center justify-between p-3">
          <h1 className="text-lg font-medium">We use cookies</h1>
          <CookieIcon className="h-[1.2rem] w-[1.2rem]" />
        </div>
        <div className="p-3 -mt-2">
          <p className="text-sm text-left text-muted-foreground">{body}</p>
        </div>
        <div className="p-3 flex items-center gap-2 mt-2 border-t">
          <Button onClick={acceptClick} className="w-full h-9 rounded-full">
            {acceptLabel.toLowerCase()}
          </Button>
          <Button
            onClick={declineClick}
            className="w-full h-9 rounded-full"
            variant="outline"
          >
            {declineLabel.toLowerCase()}
          </Button>
        </div>
      </div>
    </div>
  );
}
