/**
 * Generate llms.txt, llms-full.txt, and per-page markdown files from the
 * Hatchet documentation.
 *
 * This script reads the MDX documentation pages, resolves Snippet references
 * to inline code, expands UniversalTabs into labeled language sections, and
 * converts JSX components to plain Markdown.
 *
 * Usage:
 *   tsx scripts/generate-llms.ts                          # all languages
 *   tsx scripts/generate-llms.ts --languages python       # Python only
 *   tsx scripts/generate-llms.ts --languages python,typescript
 */

import fs from "node:fs";
import path from "node:path";
import { snippets } from "../lib/generated/snippets/index.js";

// ---------------------------------------------------------------------------
// Paths
// ---------------------------------------------------------------------------
const SCRIPT_DIR = path.dirname(new URL(import.meta.url).pathname);
const DOCS_ROOT = path.resolve(SCRIPT_DIR, "..");
const PAGES_DIR = path.join(DOCS_ROOT, "pages");
const OUTPUT_DIR = path.join(DOCS_ROOT, "public");

const DOCS_BASE_URL = "https://docs.hatchet.run";

const LANGUAGE_EXTENSIONS: Record<string, string> = {
  python: "python",
  typescript: "typescript",
  go: "go",
};

const TAB_LABEL_TO_LANG: Record<string, string> = {
  python: "python",
  typescript: "typescript",
  go: "go",
};

// ---------------------------------------------------------------------------
// Snippet resolution
// ---------------------------------------------------------------------------
type SnippetNode = Record<string, any>;

function resolveSnippetPath(
  tree: SnippetNode,
  dotpath: string,
): SnippetNode | null {
  let cleaned = dotpath;
  if (cleaned.startsWith("snippets.")) {
    cleaned = cleaned.slice("snippets.".length);
  }
  const parts = cleaned.split(".");
  let current: any = tree;
  for (const part of parts) {
    if (current && typeof current === "object" && part in current) {
      current = current[part];
    } else {
      return null;
    }
  }
  if (current && typeof current === "object" && "content" in current) {
    return current as SnippetNode;
  }
  return null;
}

// ---------------------------------------------------------------------------
// _meta.js parsing
// ---------------------------------------------------------------------------
interface DocPage {
  title: string;
  slug: string;
  href: string;
  filepath: string;
  section: string;
}

/**
 * Parse a _meta.js file into a plain object.
 *
 * **Limitations:** This uses regex to convert simple JS object literals to
 * JSON. It only supports _meta.js files that export a plain object with:
 *   - Simple unquoted or quoted string keys (no computed `[expr]` keys)
 *   - String or plain-object values (no function calls, template literals,
 *     spread operators, or variable references)
 *   - No inline or block comments
 *
 * If your _meta.js file uses any of these unsupported constructs, either
 * simplify it or extend this parser (e.g. with @babel/parser + eval).
 */
function parseMetaJs(filepath: string): Record<string, any> {
  const raw = fs.readFileSync(filepath, "utf-8");
  let content = raw.replace("export default ", "");
  // Quote unquoted object keys for JSON parsing
  const pattern = /^(\s*)([a-zA-Z_$][a-zA-Z0-9_$-]*)\s*:/gm;
  content = content.replace(pattern, '$1"$2":');
  // Apply twice to catch keys that were adjacent
  content = content.replace(pattern, '$1"$2":');
  // Remove trailing commas before closing braces
  content = content.replace(/,(\s*\n?\s*})(\s*);?/g, "$1");
  // Strip trailing semicolon from export default {...};
  content = content.replace(/\s*;\s*$/, "");

  try {
    return JSON.parse(content);
  } catch (err) {
    const message =
      err instanceof Error ? err.message : String(err);
    throw new Error(
      `Failed to parse _meta.js at ${filepath}: ${message}.\n` +
        `The regex-based parser only supports simple object literals ` +
        `(no computed keys, spread operators, comments, or expressions). ` +
        `Simplify the file or switch to a proper JS parser.\n` +
        `--- transformed content ---\n${content}`,
    );
  }
}

function isDocPage(key: string, value: any): boolean {
  if (key.trim().startsWith("--")) return false;
  if (key.trim().startsWith("_")) return false;
  if (typeof value === "string") return true;
  if (typeof value === "object" && value !== null) {
    if (value.display === "hidden") return false;
    if ("title" in value) return true;
  }
  return false;
}

function extractTitle(value: any): string {
  if (typeof value === "string") return value;
  if (typeof value === "object" && value !== null && "title" in value)
    return value.title;
  return "";
}

function collectPages(): DocPage[] {
  const pages: DocPage[] = [];

  const rootMetaPath = path.join(PAGES_DIR, "_meta.js");
  if (!fs.existsSync(rootMetaPath)) return pages;

  const rootMeta = parseMetaJs(rootMetaPath);
  const sectionOrder = Object.keys(rootMeta).filter(
    (k) => !k.startsWith("_"),
  );

  for (const sectionKey of sectionOrder) {
    const sectionDir = path.join(PAGES_DIR, sectionKey);
    const sectionMetaPath = path.join(sectionDir, "_meta.js");

    const sectionValue = rootMeta[sectionKey] ?? {};
    const sectionTitle =
      typeof sectionValue === "object"
        ? extractTitle(sectionValue)
        : sectionKey;

    if (!fs.existsSync(sectionMetaPath)) {
      const mdxPath = path.join(PAGES_DIR, sectionKey + ".mdx");
      if (fs.existsSync(mdxPath)) {
        pages.push({
          title: sectionTitle || sectionKey,
          slug: sectionKey,
          href: `${DOCS_BASE_URL}/${sectionKey}`,
          filepath: mdxPath,
          section: sectionTitle || sectionKey,
        });
      }
      continue;
    }

    const sectionMeta = parseMetaJs(sectionMetaPath);
    for (const [pageKey, pageValue] of Object.entries(sectionMeta)) {
      if (!isDocPage(pageKey, pageValue)) continue;

      const title = extractTitle(pageValue);
      let mdxPath = path.join(sectionDir, pageKey + ".mdx");

      if (!fs.existsSync(mdxPath)) {
        mdxPath = path.join(sectionDir, pageKey, "index.mdx");
      }
      if (!fs.existsSync(mdxPath)) continue;

      const href = `${DOCS_BASE_URL}/${sectionKey}/${pageKey}`;

      pages.push({
        title,
        slug: pageKey,
        href,
        filepath: mdxPath,
        section: sectionTitle || sectionKey,
      });
    }
  }

  return pages;
}

// ---------------------------------------------------------------------------
// MDX -> Markdown conversion
// ---------------------------------------------------------------------------
function stripImportLines(text: string): string {
  const lines = text.split("\n");
  const result: string[] = [];
  let inImports = true;
  for (const line of lines) {
    if (inImports) {
      const stripped = line.trim();
      if (stripped.startsWith("import ") || stripped === "") continue;
      inImports = false;
    }
    result.push(line);
  }
  return result.join("\n");
}

function stripJsxComments(text: string): string {
  return text.replace(/\{\/\*[\s\S]*?\*\/\}/g, "");
}

function resolveSnippets(
  text: string,
  snippetTree: SnippetNode,
  languages: string[] | null,
): string {
  const pattern = /<Snippet\s+src=\{([\s\S]*?)\}\s*\/>/g;
  return text.replace(pattern, (_match, rawPath: string) => {
    const dotpath = rawPath.replace(/\s+/g, "").trim();
    const snippet = resolveSnippetPath(snippetTree, dotpath);
    if (!snippet) return `<!-- snippet not found: ${dotpath} -->`;

    const lang = snippet.language ?? "";
    if (languages && !languages.includes(lang)) return "";

    const langExt = LANGUAGE_EXTENSIONS[lang] ?? lang;
    const code = (snippet.content ?? "").trimEnd();
    return `\`\`\`${langExt}\n${code}\n\`\`\``;
  });
}

function convertCallouts(text: string): string {
  const pattern = /<Callout\s+type=["'](\w+)["']\s*>([\s\S]*?)<\/Callout>/g;
  return text.replace(pattern, (_match, calloutType: string, content: string) => {
    const label = calloutType.charAt(0).toUpperCase() + calloutType.slice(1);
    const trimmed = content.trim();
    const lines = trimmed.split("\n");
    if (lines.length === 1) {
      return `> **${label}:** ${trimmed}`;
    }
    return (
      `> **${label}:** ${lines[0]}\n` +
      lines
        .slice(1)
        .map((l) => (l.trim() ? `> ${l}` : ">"))
        .join("\n")
    );
  });
}

// ---------------------------------------------------------------------------
// Tab expansion
// ---------------------------------------------------------------------------
function dedentTabContent(text: string): string {
  const lines = text.split("\n");
  let inFence = false;
  // Use a boolean array instead of Set to avoid es5 iteration issues
  const isProseLine: boolean[] = new Array(lines.length).fill(false);

  for (let i = 0; i < lines.length; i++) {
    const stripped = lines[i].trimStart();
    if (stripped.startsWith("```")) {
      inFence = !inFence;
      isProseLine[i] = true;
      continue;
    }
    if (!inFence) {
      isProseLine[i] = true;
    }
  }

  let minIndent: number | null = null;
  for (let i = 0; i < lines.length; i++) {
    if (!isProseLine[i]) continue;
    const line = lines[i];
    const stripped = line.trim();
    if (!stripped) continue;
    if (stripped.startsWith("<") || stripped.startsWith("{/*")) continue;
    const indent = line.length - line.trimStart().length;
    if (indent === 0) continue;
    if (minIndent === null || indent < minIndent) {
      minIndent = indent;
    }
  }

  if (!minIndent) return text;

  const result: string[] = [];
  for (let i = 0; i < lines.length; i++) {
    if (
      isProseLine[i] &&
      lines[i].length >= minIndent &&
      lines[i].slice(0, minIndent).trim() === ""
    ) {
      result.push(lines[i].slice(minIndent));
    } else {
      result.push(lines[i]);
    }
  }
  return result.join("\n");
}

function extractTabContents(
  inner: string,
  items: string[],
): [string, string][] {
  const result: [string, string][] = [];
  let tabIdx = 0;
  let pos = 0;

  while (pos < inner.length) {
    const openMatch = inner.slice(pos).match(/<Tabs\.Tab(?:\s+[^>]*)?>/);
    if (!openMatch || openMatch.index === undefined) break;

    const start = pos + openMatch.index + openMatch[0].length;
    let depth = 1;
    let scan = start;

    while (scan < inner.length && depth > 0) {
      const remaining = inner.slice(scan);
      const nextOpen = remaining.match(/<Tabs\.Tab(?:\s+[^>]*)?>/);
      const nextClose = remaining.match(/<\/Tabs\.Tab>/);

      if (!nextClose || nextClose.index === undefined) break;

      if (
        nextOpen &&
        nextOpen.index !== undefined &&
        nextOpen.index < nextClose.index
      ) {
        depth++;
        scan += nextOpen.index + nextOpen[0].length;
      } else {
        depth--;
        if (depth === 0) {
          let content = inner.slice(start, scan + nextClose.index);
          content = dedentTabContent(content);
          const label =
            tabIdx < items.length ? items[tabIdx] : `Tab ${tabIdx + 1}`;
          result.push([label, content]);
          tabIdx++;
          scan += nextClose.index + nextClose[0].length;
        } else {
          scan += nextClose.index + nextClose[0].length;
        }
      }
    }

    pos = scan;
  }

  return result;
}

function expandUniversalTabs(
  text: string,
  languages: string[] | null,
): string {
  const pattern =
    /<UniversalTabs\s+items=\{(\[[^\]]*\])\}(?:\s+optionKey=["']([^"']*)["'])?\s*>((?:(?!<UniversalTabs)[\s\S])*?)<\/UniversalTabs>/g;

  function processTabsBlock(
    _match: string,
    itemsStr: string,
    optionKey: string | undefined,
    inner: string,
  ): string {
    let items = itemsStr.match(/"([^"]*)"/g)?.map((s) => s.slice(1, -1)) ?? [];
    if (items.length === 0) {
      items = itemsStr.match(/'([^']*)'/g)?.map((s) => s.slice(1, -1)) ?? [];
    }

    const isLanguageTabs = !optionKey || optionKey === "language";
    const tabContents = extractTabContents(inner, items);

    const parts: string[] = [];
    for (const [label, content] of tabContents) {
      const langKey = TAB_LABEL_TO_LANG[label.toLowerCase()];

      if (isLanguageTabs && langKey && languages && !languages.includes(langKey))
        continue;

      parts.push(`#### ${label}\n\n${content.trim()}`);
    }

    return parts.join("\n\n");
  }

  // Repeatedly process innermost first (handles nesting)
  let prev: string | null = null;
  while (prev !== text) {
    prev = text;
    text = text.replace(pattern, processTabsBlock);
  }

  return text;
}

function expandStandaloneTabs(text: string): string {
  const pattern =
    /<Tabs\s+items=\{(\[[\s\S]*?\])\}\s*>([\s\S]*?)<\/Tabs>/g;

  return text.replace(pattern, (_match, itemsStr: string, inner: string) => {
    let items = itemsStr.match(/"([^"]*)"/g)?.map((s) => s.slice(1, -1)) ?? [];
    if (items.length === 0) {
      items = itemsStr.match(/'([^']*)'/g)?.map((s) => s.slice(1, -1)) ?? [];
    }

    const tabContents = extractTabContents(inner, items);
    const parts: string[] = [];
    for (const [label, content] of tabContents) {
      parts.push(`#### ${label}\n\n${content.trim()}`);
    }
    return parts.join("\n\n");
  });
}

// ---------------------------------------------------------------------------
// Other component converters
// ---------------------------------------------------------------------------
function convertSteps(text: string): string {
  text = text.replace(/<Steps\s*\/?>/g, "");
  text = text.replace(/<\/Steps>/g, "");
  return text;
}

function convertCards(text: string): string {
  text = text.replace(/<Cards\s*\/?>/g, "");
  text = text.replace(/<\/Cards>/g, "");

  text = text.replace(
    /<Card\s+([\s\S]*?)(?:>([\s\S]*?)<\/Card>|\/>)/g,
    (_match, attrs: string, content?: string) => {
      const titleMatch = attrs.match(/title=["']([^"']*)["']/);
      const hrefMatch = attrs.match(/href=["']([^"']*)["']/);
      const title = titleMatch?.[1] ?? "";
      const href = hrefMatch?.[1] ?? "";
      const trimContent = content?.trim() ?? "";

      if (href) {
        return `- [${title}](${href})${trimContent ? ": " + trimContent : ""}`;
      }
      return `- **${title}**${trimContent ? ": " + trimContent : ""}`;
    },
  );
  return text;
}

function convertFileTree(text: string): string {
  function walkFileTree(
    content: string,
    lines: string[],
    depth: number,
  ): void {
    const folderPattern =
      /<FileTree\.Folder\s+name=["']([^"']*)["'][^>]*>([\s\S]*?)<\/FileTree\.Folder>/g;
    let folderMatch: RegExpExecArray | null;
    while ((folderMatch = folderPattern.exec(content)) !== null) {
      lines.push("  ".repeat(depth) + folderMatch[1] + "/");
      walkFileTree(folderMatch[2], lines, depth + 1);
    }
    const filePattern =
      /<FileTree\.File\s+name=["']([^"']*)["'][^>]*\s*\/>/g;
    let fileMatch: RegExpExecArray | null;
    while ((fileMatch = filePattern.exec(content)) !== null) {
      lines.push("  ".repeat(depth) + fileMatch[1]);
    }
  }

  return text.replace(
    /<FileTree>([\s\S]*?)<\/FileTree>/g,
    (_match, inner: string) => {
      const lines: string[] = [];
      walkFileTree(inner, lines, 0);
      return "```\n" + lines.join("\n") + "\n```";
    },
  );
}

function stripJsxComponents(text: string): string {
  // Self-closing JSX tags
  text = text.replace(/<[A-Z]\w*(?:\.\w+)*\s*[^>]*\/\s*>/g, "");
  // Opening/closing JSX tags
  text = text.replace(/<\/?[A-Z]\w*(?:\.\w+)*\s*[^>]*>/g, "");
  return text;
}

function resolveMdxComponentImports(
  text: string,
  filepath: string,
  snippetTree: SnippetNode,
  languages: string[] | null,
  depth: number = 0,
): string {
  if (depth > 10) {
    console.warn(
      `[generate-llms] resolveMdxComponentImports: recursion depth limit ` +
        `(10) reached while processing "${filepath}". This likely indicates ` +
        `circular MDX imports. The remaining component references will not ` +
        `be resolved.`,
    );
    return text;
  }

  const mdxImportPattern =
    /import\s+(\w+)\s+from\s+["']([^"']*\.mdx)["']/g;

  // Collect all MDX component imports first
  const imports: Array<{ componentName: string; relPath: string }> = [];
  let importMatch: RegExpExecArray | null;
  while ((importMatch = mdxImportPattern.exec(text)) !== null) {
    imports.push({
      componentName: importMatch[1],
      relPath: importMatch[2],
    });
  }

  for (const imp of imports) {
    const importedFilePath = path.resolve(path.dirname(filepath), imp.relPath);
    if (!fs.existsSync(importedFilePath)) {
      // Fall back to a comment if the file can't be found
      text = text.replace(
        new RegExp(`<${imp.componentName}\\s*/\\s*>`, "g"),
        `<!-- Could not resolve ${imp.relPath} -->`,
      );
      continue;
    }

    // Read the imported MDX and recursively convert it
    const importedRaw = fs.readFileSync(importedFilePath, "utf-8");
    const importedMd = convertMdxToMarkdown(
      importedRaw,
      snippetTree,
      languages,
      importedFilePath,
      depth + 1,
    );

    // Replace all usages of <ComponentName /> with the inlined content
    text = text.replace(
      new RegExp(`<${imp.componentName}\\s*/\\s*>`, "g"),
      importedMd.trim(),
    );
  }

  return text;
}

function cleanBlankLines(text: string): string {
  return text.replace(/\n{4,}/g, "\n\n\n");
}

// ---------------------------------------------------------------------------
// Full pipeline
// ---------------------------------------------------------------------------
function convertMdxToMarkdown(
  content: string,
  snippetTree: SnippetNode,
  languages: string[] | null,
  filepath?: string,
  depth?: number,
): string {
  let text = content;

  if (filepath) {
    text = resolveMdxComponentImports(
      text,
      filepath,
      snippetTree,
      languages,
      depth ?? 0,
    );
  }
  text = stripImportLines(text);
  text = stripJsxComments(text);
  text = convertCallouts(text);
  text = resolveSnippets(text, snippetTree, languages);
  text = expandUniversalTabs(text, languages);
  text = expandStandaloneTabs(text);
  text = convertSteps(text);
  text = convertCards(text);
  text = convertFileTree(text);
  text = stripJsxComponents(text);
  text = cleanBlankLines(text);

  return text.trim() + "\n";
}

// ---------------------------------------------------------------------------
// MiniSearch index generation
// ---------------------------------------------------------------------------
import MiniSearch from "minisearch";

import { MINISEARCH_OPTIONS } from "../lib/search-config.js";

interface SearchDoc {
  id: string;
  title: string;
  content: string;
  codeIdentifiers: string;
  pageTitle: string;
  pageRoute: string;
}

/**
 * Extract compound code identifiers from fenced code blocks in markdown.
 * Finds dotted identifiers (e.g. hatchet.task, ctx.spawn, hatchet.workflow)
 * and other notable code patterns, returning them as a space-separated string.
 */
function extractCodeIdentifiers(markdown: string): string {
  const identifiers = new Set<string>();
  const lines = markdown.split("\n");
  let inFence = false;
  let fenceMarker: string | null = null;

  for (const line of lines) {
    const trimmed = line.trimStart();
    const backtickMatch = trimmed.match(/^(`{3,})/);
    if (backtickMatch) {
      if (fenceMarker === null) {
        fenceMarker = backtickMatch[1];
        inFence = true;
      } else if (backtickMatch[1].length >= fenceMarker.length) {
        fenceMarker = null;
        inFence = false;
      }
      continue;
    }

    if (!inFence) continue;

    // Dotted identifiers: hatchet.task, ctx.spawn, hatchet.workflow, etc.
    const dottedPattern = /[a-zA-Z_]\w*(?:\.[a-zA-Z_]\w*)+/g;
    let m: RegExpExecArray | null;
    while ((m = dottedPattern.exec(line)) !== null) {
      identifiers.add(m[0].toLowerCase());
    }

    // Decorated identifiers: @hatchet.task, @hatchet.workflow
    const decoratorPattern = /@([a-zA-Z_]\w*(?:\.[a-zA-Z_]\w*)*)/g;
    while ((m = decoratorPattern.exec(line)) !== null) {
      identifiers.add(m[1].toLowerCase());
    }
  }

  return Array.from(identifiers).join(" ");
}

/**
 * Convert heading text to a URL-friendly slug (matching Nextra's anchor generation).
 */
function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "");
}

/**
 * Split markdown content into sections by h2 headings.
 * Returns an array of { heading, slug, content } objects.
 * The first element has heading="" for content before the first h2.
 */
function splitByH2(
  markdown: string,
): Array<{ heading: string; slug: string; content: string }> {
  const lines = markdown.split("\n");
  const sections: Array<{ heading: string; slug: string; content: string }> = [];
  let currentHeading = "";
  let currentSlug = "";
  let currentLines: string[] = [];
  let fenceMarker: string | null = null; // tracks the opening fence (e.g. "```" or "````")

  for (const line of lines) {
    // Track fenced code blocks so we don't split on ## inside them.
    // A fence opens with 3+ backticks and closes only when we see at
    // least the same number of backticks (CommonMark spec).
    const trimmed = line.trimStart();
    const backtickMatch = trimmed.match(/^(`{3,})/);
    if (backtickMatch) {
      if (fenceMarker === null) {
        fenceMarker = backtickMatch[1]; // open fence
      } else if (backtickMatch[1].length >= fenceMarker.length) {
        fenceMarker = null; // close fence
      }
      // else: fewer backticks than the opening fence — just content
    }

    const h2Match = fenceMarker === null && line.match(/^## (.+)$/);
    if (h2Match) {
      // Flush the previous section
      const content = currentLines.join("\n").trim();
      if (content || currentHeading) {
        sections.push({
          heading: currentHeading,
          slug: currentSlug,
          content,
        });
      }
      currentHeading = h2Match[1].trim();
      currentSlug = slugify(currentHeading);
      currentLines = [];
    } else {
      currentLines.push(line);
    }
  }

  // Flush the last section
  const content = currentLines.join("\n").trim();
  if (content || currentHeading) {
    sections.push({
      heading: currentHeading,
      slug: currentSlug,
      content,
    });
  }

  return sections;
}

function buildSearchIndex(
  pages: DocPage[],
  snippetTree: SnippetNode,
  languages: string[] | null,
): string {
  const miniSearch = new MiniSearch<SearchDoc>(MINISEARCH_OPTIONS);

  const docs: SearchDoc[] = [];
  for (const page of pages) {
    const raw = fs.readFileSync(page.filepath, "utf-8");
    const md = convertMdxToMarkdown(raw, snippetTree, languages, page.filepath);
    const urlPath = page.href.replace(DOCS_BASE_URL + "/", "");
    const pageRoute = `hatchet://docs/${urlPath}`;

    const sections = splitByH2(md);

    for (const section of sections) {
      if (!section.content.trim()) continue;

      const id = section.slug
        ? `${pageRoute}#${section.slug}`
        : pageRoute;
      const title = section.heading || page.title;

      docs.push({
        id,
        title,
        content: section.content,
        codeIdentifiers: extractCodeIdentifiers(section.content),
        pageTitle: page.title,
        pageRoute,
      });
    }
  }

  miniSearch.addAll(docs);
  return JSON.stringify(miniSearch);
}

// ---------------------------------------------------------------------------
// Output generation
// ---------------------------------------------------------------------------
function generateLlmsTxt(pages: DocPage[]): string {
  const lines: string[] = [
    "# Hatchet Documentation",
    "",
    "> Hatchet is a distributed task queue and workflow engine for modern " +
      "applications. It provides durable execution, concurrency control, " +
      "rate limiting, and observability for background tasks and workflows " +
      "in Python, TypeScript, and Go.",
    "",
  ];

  let currentSection = "";
  for (const page of pages) {
    if (page.section !== currentSection) {
      currentSection = page.section;
      lines.push(`## ${currentSection}`);
      lines.push("");
    }
    lines.push(`- [${page.title}](${page.href})`);
  }

  lines.push("");
  return lines.join("\n");
}

function generateLlmsFullTxt(
  pages: DocPage[],
  snippetTree: SnippetNode,
  languages: string[] | null,
): string {
  const parts: string[] = [
    "# Hatchet Documentation",
    "",
    "> Hatchet is a distributed task queue and workflow engine for modern " +
      "applications. It provides durable execution, concurrency control, " +
      "rate limiting, and observability for background tasks and workflows " +
      "in Python, TypeScript, and Go.",
    "",
  ];

  for (const page of pages) {
    const raw = fs.readFileSync(page.filepath, "utf-8");
    const md = convertMdxToMarkdown(raw, snippetTree, languages, page.filepath);
    parts.push(`---\n\n<!-- Source: ${page.href} -->\n`);
    parts.push(md);
    parts.push("");
  }

  return parts.join("\n");
}

function generatePerPageMarkdown(
  pages: DocPage[],
  snippetTree: SnippetNode,
  languages: string[] | null,
): void {
  const llmsDir = path.join(OUTPUT_DIR, "llms");

  for (const page of pages) {
    const raw = fs.readFileSync(page.filepath, "utf-8");
    const md = convertMdxToMarkdown(raw, snippetTree, languages, page.filepath);

    const urlPath = page.href.replace(DOCS_BASE_URL + "/", "");
    const outPath = path.join(llmsDir, urlPath + ".md");
    fs.mkdirSync(path.dirname(outPath), { recursive: true });
    fs.writeFileSync(outPath, md);

    // For index pages (e.g. home/index), also write at the section root
    // (e.g. home.md) so that /llms/home.md resolves correctly — Next.js
    // router.pathname for section roots is "/home", not "/home/index".
    if (page.slug === "index") {
      const sectionPath = urlPath.replace(/\/index$/, "");
      const sectionOutPath = path.join(llmsDir, sectionPath + ".md");
      fs.writeFileSync(sectionOutPath, md);
    }
  }

  console.log(
    `  Wrote ${pages.length} per-page markdown files to ${llmsDir}/`,
  );
}

// ---------------------------------------------------------------------------
// CLI & main
// ---------------------------------------------------------------------------
function parseArgs(): string[] | null {
  const idx = process.argv.indexOf("--languages");
  if (idx === -1 || idx + 1 >= process.argv.length) return null;

  const raw = process.argv[idx + 1];
  const langs = raw.split(",").map((l) => l.trim().toLowerCase());
  const valid = Object.keys(LANGUAGE_EXTENSIONS);
  for (const lang of langs) {
    if (!valid.includes(lang)) {
      console.error(
        `Unknown language: ${lang}. Valid: ${valid.sort().join(", ")}`,
      );
      process.exit(1);
    }
  }
  return langs;
}

function main(): void {
  const languages = parseArgs();

  console.log("Loading snippets...");
  const snippetTree = snippets as unknown as SnippetNode;

  console.log("Collecting pages from _meta.js files...");
  const pages = collectPages();
  console.log(`  Found ${pages.length} pages`);

  console.log("Generating llms.txt...");
  const llmsTxt = generateLlmsTxt(pages);

  console.log("Generating llms-full.txt...");
  const llmsFullTxt = generateLlmsFullTxt(pages, snippetTree, languages);

  console.log("Generating per-page markdown files...");
  generatePerPageMarkdown(pages, snippetTree, languages);

  console.log("Building MiniSearch index...");
  const searchIndexJson = buildSearchIndex(pages, snippetTree, languages);

  fs.mkdirSync(OUTPUT_DIR, { recursive: true });

  const llmsTxtPath = path.join(OUTPUT_DIR, "llms.txt");
  fs.writeFileSync(llmsTxtPath, llmsTxt);
  console.log(`  Wrote ${llmsTxtPath} (${llmsTxt.length} bytes)`);

  const llmsFullPath = path.join(OUTPUT_DIR, "llms-full.txt");
  fs.writeFileSync(llmsFullPath, llmsFullTxt);
  console.log(`  Wrote ${llmsFullPath} (${llmsFullTxt.length} bytes)`);

  const searchIndexPath = path.join(OUTPUT_DIR, "llms-search-index.json");
  fs.writeFileSync(searchIndexPath, searchIndexJson);
  console.log(
    `  Wrote ${searchIndexPath} (${searchIndexJson.length} bytes)`,
  );

  if (languages) {
    console.log(`  Languages: ${languages.join(", ")}`);
  } else {
    console.log("  Languages: all");
  }

  console.log("Done!");
}

main();
