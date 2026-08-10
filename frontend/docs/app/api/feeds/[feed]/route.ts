import { FEEDS } from "@/lib/feeds/registry";

const SITE = "https://docs.hatchet.run";

export async function GET(
  _req: Request,
  { params }: { params: Promise<{ feed: string }> },
) {
  const { feed } = await params;
  const buildFeed = FEEDS[feed];
  if (!buildFeed) {
    return Response.json({ error: `Unknown feed: ${feed}` }, { status: 404 });
  }

  // TODO(gregfurman): Ensure CDN doesn't cache XML feed for too long.
  try {
    const xml = buildFeed({ site: SITE, feedUrl: `${SITE}/api/feeds/${feed}` });
    return new Response(xml, {
      headers: {
        "Content-Type": "application/rss+xml; charset=utf-8",
        "Cache-Control": "public, s-maxage=3600, stale-while-revalidate=86400",
      },
    });
  } catch (error) {
    console.error(`Failed to build ${feed} feed:`, error);
    return Response.json({ error: "Failed to build feed" }, { status: 500 });
  }
}
