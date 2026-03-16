/**
 * Keywords component — visually hidden, but indexed with high weight in
 * the MiniSearch search index.  Place at the bottom of a docs page to
 * boost discoverability for synonyms and alternate phrasings.
 *
 * Usage:
 *   <Keywords keywords="overview, introduction, concepts, use cases" />
 *
 * The `generate-llms` script extracts the `keywords` prop and stores it in
 * a dedicated high-boost field in the search index.  The component itself
 * renders a visually-hidden element so the text is never seen by readers.
 */
export default function Keywords({ keywords }: { keywords: string }) {
  return (
    <span style={{ display: "none" }} aria-hidden="true">
      {keywords}
    </span>
  );
}
