import React from "react";
import { useRouter } from "next/router";
import { useLanguage } from "../context/LanguageContext";
import {
  DOC_LANGUAGES,
  DEFAULT_LANGUAGE,
  LOGO_PATHS,
} from "@/lib/docs-languages";

function ThemedIcon({ src }: { src: string }) {
  return (
    <span
      style={
        {
          display: "inline-block",
          width: 14,
          height: 14,
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

function resolveSelected(lang: string) {
  const exact = DOC_LANGUAGES.find((l) => l === lang);
  if (exact) return exact;
  const lower = lang.toLowerCase();
  const match = DOC_LANGUAGES.find((l) => l.toLowerCase() === lower);
  return match ?? DEFAULT_LANGUAGE;
}

export default function LanguageSwitcher() {
  const router = useRouter();
  const basePath = router.basePath || "";
  const { selectedLanguage, setSelectedLanguage } = useLanguage();
  const current = resolveSelected(selectedLanguage);

  return (
    <div className="language-switcher mt-6 mb-2">
      <p className="mb-2 text-sm text-[hsl(var(--muted-foreground))]">
        Customize your docs experience — choose your preferred language for code examples:
      </p>
      <div
        className="flex flex-wrap gap-1 rounded-lg border border-[hsl(var(--border))] bg-[hsl(var(--muted)/0.5)] p-1"
        role="tablist"
        aria-label="Choose documentation language"
      >
        {DOC_LANGUAGES.map((lang) => {
        const filename = LOGO_PATHS[lang];
        const isSelected = current === lang;
        return (
          <button
            key={lang}
            type="button"
            role="tab"
            aria-selected={isSelected}
            onClick={() => setSelectedLanguage(lang)}
            className={`
              inline-flex items-center gap-1.5 rounded-md px-3 py-2 text-sm font-medium
              transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))]
              ${
                isSelected
                  ? "bg-[hsl(var(--background))] text-[hsl(var(--foreground))] shadow-sm"
                  : "text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))]"
              }
            `}
          >
            {filename ? (
              <ThemedIcon src={`${basePath}/${filename}`.replace(/\/+/g, "/")} />
            ) : null}
            {lang}
          </button>
        );
        })}
      </div>
    </div>
  );
}
