import fs from "node:fs";
import path from "node:path";
import {
  renderRSS,
  type FeedItem,
  type Channel,
  FeedContext,
} from "@/lib/feeds/rss";

interface Cookbook {
  slug: string;
  title: string;
  section: string;
}

const COOKBOOKS_DIR = path.join(process.cwd(), "content/docs/cookbooks");

function readFrontmatterTitle(mdxPath: string): string | null {
  const src = fs.readFileSync(mdxPath, "utf-8");
  const fm = /^---\n([\s\S]*?)\n---/.exec(src);
  const line = fm?.[1].split("\n").find((l) => /^title\s*:/.test(l));
  if (!line) return null;
  return line
    .replace(/^title\s*:\s*/, "")
    .replace(/^["']|["']$/g, "")
    .replace(/\\"/g, '"');
}

function extractCookbooks(): Cookbook[] {
  const meta = JSON.parse(
    fs.readFileSync(path.join(COOKBOOKS_DIR, "meta.json"), "utf-8"),
  ) as { pages?: string[] };
  const out: Cookbook[] = [];
  let section = "";

  for (const key of meta.pages ?? []) {
    const sep = /^---(.*)---$/.exec(key);
    if (sep) {
      section = sep[1];
    } else if (key !== "index") {
      const mdxPath = path.join(COOKBOOKS_DIR, `${key}.mdx`);
      if (!fs.existsSync(mdxPath)) continue;
      out.push({
        slug: key,
        title: readFrontmatterTitle(mdxPath) ?? key,
        section,
      });
    }
  }

  return out;
}

export function buildCookbooksFeed({ site, feedUrl }: FeedContext): string {
  const page = `${site}/cookbooks`;

  const items: FeedItem[] = extractCookbooks().map((c) => ({
    title: c.section ? `${c.section}: ${c.title}` : c.title,
    link: `${page}/${c.slug}`,
    image: `${site}/og.png`,
  }));

  const feed: Channel = {
    title: "Hatchet Cookbooks",
    link: page,
    self: feedUrl,
    description: "Guides and recipes for building with Hatchet",
    language: "en",
    logo: `${site}/logo.png`,
  };

  return renderRSS(feed, items);
}
