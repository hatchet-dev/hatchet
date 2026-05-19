/**
 * Single source of truth for documentation languages and their metadata.
 * Used by SidebarLanguageButton, LanguageSwitcher, UniversalTabs, LanguageContext,
 * PackageManagerInstall, InstallCommand, and integration tab components.
 */

export const DOC_LANGUAGES = ["Python", "Typescript", "Go", "Ruby"] as const;
export type DocLanguage = (typeof DOC_LANGUAGES)[number];

export const DEFAULT_LANGUAGE: DocLanguage = "Python";

/** Logo filename in public/ for each language. Includes aliases used in UniversalTabs. */
export const LOGO_PATHS: Record<string, string> = {
  Python: "python-logo.svg",
  "Python-Sync": "python-logo.svg",
  "Python-Async": "python-logo.svg",
  Typescript: "typescript-logo.svg",
  Go: "go-logo.svg",
  Ruby: "ruby-logo.svg",
};

/** Package manager options for languages that support choice. Null = fixed tool. */
export const PACKAGE_MANAGERS: Record<
  DocLanguage,
  readonly string[] | { fixed: string }
> = {
  Python: ["pip", "poetry", "uv"],
  Typescript: ["npm", "pnpm", "yarn"],
  Go: ["go get"],
  Ruby: ["bundle"],
};

export function getPackageManagers(
  lang: DocLanguage
): readonly string[] | null {
  const pm = PACKAGE_MANAGERS[lang];
  if (Array.isArray(pm)) return pm;
  return null;
}

export function getFixedPackageManagerMessage(lang: DocLanguage): string | null {
  const pm = PACKAGE_MANAGERS[lang];
  if (pm && typeof pm === "object" && "fixed" in pm) return pm.fixed;
  return null;
}
