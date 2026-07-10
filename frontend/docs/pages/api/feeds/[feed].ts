import type { NextApiRequest, NextApiResponse } from "next";
import { FEEDS } from "@/lib/feeds/registry";

const SITE = "https://docs.hatchet.run";

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse,
): void {
  if (req.method !== "GET") {
    res.status(405).json({ error: "Method not allowed" });
    return;
  }

  const { feed } = req.query;
  const buildFeed = typeof feed === "string" ? FEEDS[feed] : undefined;
  if (!buildFeed) {
    res.status(404).json({ error: `Unknown feed: ${feed}` });
    return;
  }

  // TODO(gregfurman): Ensure CDN doesn't cache XML feed for too long.
  try {
    const xml = buildFeed({ site: SITE, feedUrl: `${SITE}/api/feeds/${feed}` });
    res.setHeader("Content-Type", "application/rss+xml; charset=utf-8");
    res.setHeader(
      "Cache-Control",
      "public, s-maxage=3600, stale-while-revalidate=86400",
    );
    res.send(xml);
  } catch (error) {
    console.error(`Failed to build ${feed} feed:`, error);
    res.status(500).json({ error: "Failed to build feed" });
  }
}
