import type { StructuredData } from "fumadocs-core/mdx-plugins";

export type SeoPageData = {
  title: string;
  description?: string;
  seoTitle?: string;
  structuredData: StructuredData | (() => Promise<StructuredData>);
};

const SDK_LANGUAGES: Record<string, string> = {
  python: "Python",
  typescript: "TypeScript",
  go: "Go",
  ruby: "Ruby",
};

/** "Python SDK" for /reference/python/**, undefined elsewhere. */
export function sdkLabel(slug: string[]): string | undefined {
  if (slug[0] !== "reference") return undefined;
  const language = SDK_LANGUAGES[slug[1] ?? ""];
  return language ? `${language} SDK` : undefined;
}

/**
 * The <title> for a page. `seoTitle` wins; SDK reference pages get their
 * language appended because the generators emit the same titles for every
 * language ("Client", "Context", "Overview", ...).
 */
export function pageTitle(slug: string[], data: SeoPageData): string {
  if (data.seoTitle) return data.seoTitle;
  const sdk = sdkLabel(slug);
  if (!sdk) return data.title;
  if (slug.length === 2) return `${sdk} Reference`;
  return `${data.title} (${sdk})`;
}

const MAX_DESCRIPTION = 160;
const MIN_SENTENCE_CUT = 70;

/** Reduce inline markdown left in fumadocs' structured text to plain words. */
function stripMarkdown(text: string): string {
  return text
    .replace(/!\[[^\]]*\]\([^)]*\)/g, "")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/`([^`]*)`/g, "$1")
    .replace(/(\*\*|__)(.+?)\1/g, "$2")
    .replace(/(^|\s)[*_]([^*_\s][^*_]*?)[*_](?=[\s.,;:!?]|$)/g, "$1$2")
    .replace(/^>\s*/g, "")
    .replace(/#+\s*/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

const SENTENCE = /[.!?](\s|$)/;

/** Trim body text to a meta-description length, preferring a sentence boundary. */
export function summarize(text: string, max = MAX_DESCRIPTION): string {
  const clean = text.replace(/\s+/g, " ").trim();
  if (clean.length <= max) return clean;
  const window = clean.slice(0, max);
  const sentenceEnd = Math.max(
    window.lastIndexOf(". "),
    window.lastIndexOf("? "),
    window.lastIndexOf("! "),
  );
  if (sentenceEnd >= MIN_SENTENCE_CUT) return window.slice(0, sentenceEnd + 1);
  const wordEnd = window.lastIndexOf(" ");
  return `${window.slice(0, wordEnd > 0 ? wordEnd : max).replace(/[,;:]$/, "")}...`;
}

/**
 * Meta description: the frontmatter `description` when present, otherwise
 * the opening prose of the page (from fumadocs' structured data) trimmed to
 * a search-snippet length.
 */
export async function pageDescription(
  data: SeoPageData,
  sdk?: string,
): Promise<string | undefined> {
  if (data.description) return data.description;
  const structured =
    typeof data.structuredData === "function"
      ? await data.structuredData()
      : data.structuredData;
  const blocks = (structured?.contents ?? [])
    .map((block) => stripMarkdown(block.content))
    .filter(Boolean);
  // Prefer real sentences over fragments such as "Bases: BaseRestClient" or
  // "Methods: Name Description" that generated reference pages open with.
  const prose = blocks.filter((b) => b.length >= 40 && SENTENCE.test(b));
  let text = "";
  for (const block of prose.length ? prose : blocks) {
    text = text ? `${text} ${block}` : block;
    if (text.length >= MAX_DESCRIPTION) break;
    if (text.length >= 60 && /[.!?]$/.test(text)) break;
  }
  if (!text) return undefined;
  // The same doc comment is generated for every SDK, so name the language.
  return sdk
    ? `${sdk}: ${summarize(text, MAX_DESCRIPTION - sdk.length - 2)}`
    : summarize(text);
}
