
export interface FeedContext {
  site: string;
  feedUrl: string;
}

export interface Channel {
  title: string;
  link: string;
  self: string;
  description: string;
  language: string;
  logo?: string;
  lastBuildDate?: string;
}

export interface FeedItem {
  title: string;
  link: string;
  description?: string;
  /** RFC-822 date. */
  pubDate?: string;
  /** Thumbnail URL, rendered as <media:thumbnail>. */
  thumbnail?: string;
}

/** Escape XML special characters so titles/URLs with `&`, `<`, etc. don't break the feed. */
function xml(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;");
}

function renderItem(item: FeedItem): string {
  const fields = [
    `<title>${xml(item.title)}</title>`,
    `<link>${xml(item.link)}</link>`,
    item.description && `<description>${xml(item.description)}</description>`,
    item.pubDate && `<pubDate>${item.pubDate}</pubDate>`,
    `<guid isPermaLink="true">${xml(item.link)}</guid>`,
    item.thumbnail &&
      `<media:thumbnail url="${xml(item.thumbnail)}" height="600" width="900"/>`,
  ].filter(Boolean);
  return `<item>\n      ${fields.join("\n      ")}\n    </item>`;
}

export function renderRSS(channel: Channel, items: FeedItem[]): string {
  const optional = [
    channel.lastBuildDate &&
      `<lastBuildDate>${channel.lastBuildDate}</lastBuildDate>`,
    channel.logo &&
      `<image>
      <url>${xml(channel.logo)}</url>
      <title>${xml(channel.title)}</title>
      <link>${xml(channel.link)}</link>
    </image>`,
  ].filter(Boolean);

  return `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"
     xmlns:atom="http://www.w3.org/2005/Atom"
     xmlns:media="http://search.yahoo.com/mrss/">
  <channel>
    <title>${xml(channel.title)}</title>
    <link>${xml(channel.link)}</link>
    <description>${xml(channel.description)}</description>
    <atom:link href="${xml(channel.self)}" rel="self" type="application/rss+xml"/>
    <language>${channel.language}</language>
    ${optional.join("\n    ")}
    ${items.map(renderItem).join("\n    ")}
  </channel>
</rss>`;
}
