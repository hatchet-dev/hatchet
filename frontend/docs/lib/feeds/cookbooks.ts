import { renderRSS, type FeedItem, type Channel, FeedContext } from "@/lib/feeds/rss";
import meta from "@/pages/cookbooks/_meta.js";

interface Cookbook {
    slug: string;
    title: string;
    section: string;
}

type MetaEntry = string | { title?: string; type?: string };

function extractCookbooks(): Cookbook[] {
    const out: Cookbook[] = [];
    let section = "";

    for (const [key, value] of Object.entries(meta as Record<string, MetaEntry>)) {
        if (typeof value === "object" && value.type === "separator") {
            section = value.title ?? "";
        } else if (key !== "index") {
            const title = typeof value === "string" ? value : value.title ?? key;
            out.push({ slug: key, title, section });
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
    }

    return renderRSS(feed, items);
}
