import { useLanguage } from "@/context/LanguageContext";
import { getPackageManagers } from "@/lib/docs-languages";
import { Tabs } from "nextra/components";
import UniversalTabs from "@/components/UniversalTabs";
import { CodeBlock } from "@/components/code/CodeBlock";

export interface PackageManagerInstallProps {
  /** Package spec per language. Omit a language to not show install (e.g. when using Callout instead). */
  packages: {
    python?: string;
    typescript?: string;
    go?: string;
    ruby?: string;
  };
  /** Optional key to scope package manager preference (default: packageManager) */
  optionKey?: string;
}

/** Renders install commands with language-specific package manager tabs (pip/poetry/uv, npm/pnpm/yarn). */
export default function PackageManagerInstall({
  packages,
  optionKey = "packageManager",
}: PackageManagerInstallProps) {
  const { selectedLanguage } = useLanguage();

  const normalizedLang =
    selectedLanguage === "TypeScript"
      ? "typescript"
      : selectedLanguage.toLowerCase();

  const pkg = packages[normalizedLang as keyof typeof packages];
  if (!pkg) return null;

  if (normalizedLang === "python") {
    const options = getPackageManagers("Python")!;
    return (
      <UniversalTabs items={[...options]} optionKey={optionKey}>
        <Tabs.Tab title="pip">
          <CodeBlock source={{ language: "bash", raw: `pip install ${pkg}` }} />
        </Tabs.Tab>
        <Tabs.Tab title="poetry">
          <CodeBlock source={{ language: "bash", raw: `poetry add ${pkg}` }} />
        </Tabs.Tab>
        <Tabs.Tab title="uv">
          <CodeBlock source={{ language: "bash", raw: `uv add ${pkg}` }} />
        </Tabs.Tab>
      </UniversalTabs>
    );
  }

  if (normalizedLang === "typescript") {
    const options = getPackageManagers("Typescript")!;
    return (
      <UniversalTabs items={[...options]} optionKey={optionKey}>
        <Tabs.Tab title="npm">
          <CodeBlock source={{ language: "bash", raw: `npm install ${pkg}` }} />
        </Tabs.Tab>
        <Tabs.Tab title="pnpm">
          <CodeBlock source={{ language: "bash", raw: `pnpm add ${pkg}` }} />
        </Tabs.Tab>
        <Tabs.Tab title="yarn">
          <CodeBlock source={{ language: "bash", raw: `yarn add ${pkg}` }} />
        </Tabs.Tab>
      </UniversalTabs>
    );
  }

  if (normalizedLang === "go") {
    return <CodeBlock source={{ language: "bash", raw: `go get ${pkg}` }} />;
  }

  if (normalizedLang === "ruby") {
    return (
      <CodeBlock source={{ language: "bash", raw: `bundle add ${pkg}` }} />
    );
  }

  return null;
}
