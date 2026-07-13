import type { FeedContext } from "@/lib/feeds/rss";
import { buildChangelogFeed } from "./changelog";
import { buildCookbooksFeed } from "./cookbooks";


export const FEEDS: Record<string, (ctx: FeedContext) => string> = {
  platform: (ctx) =>
    buildChangelogFeed(ctx, { label: "Platform", pageSlug: "platform" }),
  python: (ctx) =>
    buildChangelogFeed(ctx, { label: "Python SDK", pageSlug: "python" }),
  typescript: (ctx) =>
    buildChangelogFeed(ctx, { label: "TypeScript SDK", pageSlug: "typescript" }),
  ruby: (ctx) =>
    buildChangelogFeed(ctx, { label: "Ruby SDK", pageSlug: "ruby" }),
  cookbooks: buildCookbooksFeed,
};
