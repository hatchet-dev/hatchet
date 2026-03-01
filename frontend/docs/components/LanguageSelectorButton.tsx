import React from "react";
import { useRouter } from "next/router";
import { useLanguage } from "../context/LanguageContext";
import {
  DOC_LANGUAGES,
  DEFAULT_LANGUAGE,
  LOGO_PATHS,
  getPackageManagers,
  getFixedPackageManagerMessage,
  type DocLanguage,
} from "@/lib/docs-languages";
import { ChevronDownIcon } from "@radix-ui/react-icons";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

function ThemedIcon({ src, size = 12 }: { src: string; size?: number }) {
  return (
    <span
      style={
        {
          display: "inline-block",
          width: size,
          height: size,
          flexShrink: 0,
          backgroundColor: "currentColor",
          WebkitMaskImage: `url(${src})`,
          WebkitMaskSize: "contain",
          WebkitMaskRepeat: "no-repeat",
          WebkitMaskPosition: "center",
          maskImage: `url(${src})`,
          maskSize: "contain",
          maskRepeat: "no-repeat",
          maskPosition: "center",
        } as React.CSSProperties
      }
    />
  );
}

function resolveSelected(lang: string): DocLanguage {
  const exact = DOC_LANGUAGES.find((l) => l === lang);
  if (exact) return exact;
  const lower = lang.toLowerCase();
  const match = DOC_LANGUAGES.find((l) => l.toLowerCase() === lower);
  return match ?? DEFAULT_LANGUAGE;
}

function resolvePackageManager(lang: DocLanguage, current: string): string {
  const opts = getPackageManagers(lang);
  if (!opts) return "";
  const exact = opts.find((p) => p === current);
  if (exact) return exact;
  const lower = current.toLowerCase();
  const match = opts.find((p) => p.toLowerCase() === lower);
  return match ?? opts[0];
}

function LanguageModalContent() {
  const router = useRouter();
  const basePath = router.basePath || "";
  const {
    selectedLanguage,
    setSelectedLanguage,
    getSelectedOption,
    setSelectedOption,
  } = useLanguage();
  const current = resolveSelected(selectedLanguage);
  const pmOptions = getPackageManagers(current);
  const currentPm = pmOptions
    ? resolvePackageManager(
        current,
        getSelectedOption("packageManager") || pmOptions[0],
      )
    : null;

  return (
    <div className="grid grid-cols-1 gap-4 py-2 sm:grid-cols-2 sm:gap-6">
      <div>
        <div className="mb-2 text-xs font-medium uppercase tracking-wider text-[hsl(var(--muted-foreground))]">
          Language
        </div>
        <div className="grid gap-2">
          {DOC_LANGUAGES.map((lang) => {
            const filename = LOGO_PATHS[lang];
            const isSelected = current === lang;
            return (
              <button
                key={lang}
                type="button"
                onClick={() => setSelectedLanguage(lang)}
                className={`
                  flex items-center gap-2 rounded-md px-3 py-2.5 text-sm font-medium text-left
                  transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))]
                  ${
                    isSelected
                      ? "bg-[hsl(var(--accent))] text-[hsl(var(--accent-foreground))]"
                      : "hover:bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]"
                  }
                `}
              >
                {filename ? (
                  <ThemedIcon
                    src={`${basePath}/${filename}`.replace(/\/+/g, "/")}
                    size={14}
                  />
                ) : null}
                {lang}
              </button>
            );
          })}
        </div>
      </div>
      <div>
        <div className="mb-2 text-xs font-medium uppercase tracking-wider text-[hsl(var(--muted-foreground))]">
          Package manager
        </div>
        {pmOptions ? (
          <div className="grid gap-2">
            {pmOptions.map((pm) => {
              const isSelected = currentPm === pm;
              return (
                <button
                  key={pm}
                  type="button"
                  onClick={() => setSelectedOption("packageManager", pm)}
                  className={`
                    flex items-center gap-2 rounded-md px-3 py-2.5 text-sm font-medium text-left
                    transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))]
                    ${
                      isSelected
                        ? "bg-[hsl(var(--accent))] text-[hsl(var(--accent-foreground))]"
                        : "hover:bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]"
                    }
                  `}
                >
                  {pm}
                </button>
              );
            })}
          </div>
        ) : (
          <p className="text-sm text-[hsl(var(--muted-foreground))]">
            {getFixedPackageManagerMessage(current) ?? ""}
          </p>
        )}
      </div>
    </div>
  );
}

export function LanguageSelectorButton() {
  const router = useRouter();
  const basePath = router.basePath || "";
  const { selectedLanguage } = useLanguage();
  const current = resolveSelected(selectedLanguage);

  return (
    <div className="ml-auto">
      <Dialog>
        <DialogTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            className="h-8 gap-2 px-3.5 py-2"
            title={`Docs preferences (${current})`}
            aria-label={`Choose documentation language (${current})`}
          >
            <ThemedIcon
              src={`${basePath}/${LOGO_PATHS[current] || ""}`.replace(
                /\/+/g,
                "/",
              )}
              size={18}
            />
            <ChevronDownIcon className="h-4 w-4 opacity-70 shrink-0" />
          </Button>
        </DialogTrigger>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Docs preferences</DialogTitle>
          </DialogHeader>
          <LanguageModalContent />
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button">Save</Button>
            </DialogClose>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
