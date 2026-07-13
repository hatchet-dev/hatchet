import { renderRSS, type FeedItem, type Channel, FeedContext } from "@/lib/feeds/rss";
import fs from "node:fs";
import path from "node:path";

const RELEASE_HEADING = /^## (v\d+\.\d+\.\d+(?:-[\w.]+)?) - (\d{4}-\d{2}-\d{2})\s*$/;
const MAX_ITEMS_PER_FEED = 20;

interface Release {
  version: string;
  date: string;
  body: string
}

export interface ChangelogSource {
  label: string;
  pageSlug: string;
}

function slug(heading: string): string {
  return heading.toLowerCase().replace(/[^a-z0-9 -]/g, "").replace(/ /g, "-");
}

function extractDescription(body: string): string {
  // allows us to extract the prose from under the release header
  // up-to (but not including) the next header encountered.
  const paragraph = body
    .split(/\n\s*\n/)
    .map((p) => p.trim())
    .find((p) => p && !/^[#\-`]/.test(p)) ?? "";

  return paragraph
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/[`*_]/g, "")
    .replace(/\s+/g, " ")
    .trim();
}


function parseMarkdown(md: string): Release[] {
  const out: Release[] = [];
  let buf: string[] = [];

  const flush = () => {
    if (out.length) out[out.length - 1].body = buf.join("\n").trim();
    buf = [];
  };

  for (const line of md.split("\n")) {
    if (out.length >= MAX_ITEMS_PER_FEED) {
      break
    }

    const match = line.match(RELEASE_HEADING);
    if (match) {
      flush();
      out.push({ version: match[1], date: match[2], body: "" });
      continue
    }

    buf.push(line);

  }
  flush();

  return out;
}

export function buildChangelogFeed(
  { site, feedUrl }: FeedContext,
  { label, pageSlug }: ChangelogSource,
): string {
  const page = `${site}/reference/changelog/${pageSlug}`;
  const source = path.join(process.cwd(), "pages/reference/changelog", `${pageSlug}.mdx`);

  const items: FeedItem[] = parseMarkdown(fs.readFileSync(source, "utf-8")).map(
    ({ version, date, body }) => ({
      title: `Hatchet ${label} ${version}`,
      description: extractDescription(body),
      link: `${page}#${slug(`${version} - ${date}`)}`,
      image: `${site}/og.png`,
      pubDate: new Date(`${date}T00:00:00Z`).toUTCString(),
    }),
  );

  const feed: Channel = {
    title: `Hatchet ${label} Changelog`,
    link: page,
    self: feedUrl,
    description: `Release notes for Hatchet ${label}`,
    language: "en",
    logo: `${site}/logo.png`,
    lastBuildDate: items[0]?.pubDate,
  }

  return renderRSS(feed, items);
}
