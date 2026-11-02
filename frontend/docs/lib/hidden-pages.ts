// Pages that are published but not announced yet. They stay reachable by URL, so links
// from the dashboard keep working, but they are left out of the sitemap and marked
// noindex. The sidebar, llms.txt, the search index and the MCP catalogue only cover pages
// listed in a meta.json, so hiding a section there means unlisting it as well.
const HIDDEN_PAGE_PREFIXES = ["/v1/serverless"];

export function isHiddenPage(url: string): boolean {
  return HIDDEN_PAGE_PREFIXES.some(
    (prefix) => url === prefix || url.startsWith(`${prefix}/`),
  );
}
