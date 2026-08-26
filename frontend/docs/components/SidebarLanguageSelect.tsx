"use client";

import React from "react";
import * as Select from "@radix-ui/react-select";
import { Check, ChevronDown } from "lucide-react";
import { useLanguage } from "@/context/LanguageContext";
import {
  DOC_LANGUAGES,
  DEFAULT_LANGUAGE,
  LOGO_PATHS,
} from "@/lib/docs-languages";

function LangIcon({ src }: { src: string }) {
  return (
    <span
      aria-hidden
      style={
        {
          display: "inline-block",
          width: 15,
          height: 15,
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

function resolve(lang: string) {
  return (
    DOC_LANGUAGES.find((l) => l.toLowerCase() === lang.toLowerCase()) ??
    DEFAULT_LANGUAGE
  );
}

export function SidebarLanguageSelect() {
  const { selectedLanguage, setSelectedLanguage } = useLanguage();
  const current = resolve(selectedLanguage);

  return (
    <div className="mb-2 flex flex-col gap-1.5">
      <span className="px-1 text-[0.68rem] font-semibold uppercase tracking-wider text-fd-muted-foreground">
        SDK
      </span>
      <Select.Root value={current} onValueChange={setSelectedLanguage}>
        <Select.Trigger
          aria-label="Documentation language for code examples"
          className="inline-flex items-center gap-2 rounded-lg border border-fd-border bg-fd-secondary/40 px-2.5 py-2 text-[0.82rem] font-medium text-fd-foreground outline-none transition-colors hover:bg-fd-accent focus-visible:ring-2 focus-visible:ring-fd-ring"
        >
          <LangIcon src={`/${LOGO_PATHS[current]}`} />
          <Select.Value />
          <Select.Icon className="ms-auto text-fd-muted-foreground">
            <ChevronDown className="size-4" />
          </Select.Icon>
        </Select.Trigger>
        <Select.Portal>
          <Select.Content
            position="popper"
            sideOffset={6}
            className="z-50 min-w-[var(--radix-select-trigger-width)] overflow-hidden rounded-lg border border-fd-border bg-fd-popover text-fd-popover-foreground shadow-lg"
          >
            <Select.Viewport className="p-1">
              {DOC_LANGUAGES.map((lang) => (
                <Select.Item
                  key={lang}
                  value={lang}
                  className="relative flex cursor-pointer select-none items-center gap-2 rounded-md py-1.5 pe-8 ps-2.5 text-[0.82rem] outline-none data-[highlighted]:bg-fd-accent data-[highlighted]:text-fd-accent-foreground data-[state=checked]:text-fd-primary"
                >
                  <LangIcon src={`/${LOGO_PATHS[lang]}`} />
                  <Select.ItemText>{lang}</Select.ItemText>
                  <Select.ItemIndicator className="absolute end-2 inline-flex">
                    <Check className="size-4" />
                  </Select.ItemIndicator>
                </Select.Item>
              ))}
            </Select.Viewport>
          </Select.Content>
        </Select.Portal>
      </Select.Root>
    </div>
  );
}
