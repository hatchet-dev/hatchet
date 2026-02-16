/**
 * Shared MiniSearch configuration used at:
 *   1. Index generation time (scripts/generate-llms.ts)
 *   2. MCP server query time (pages/api/mcp.ts)
 *   3. Browser search UI (components/Search.tsx)
 *
 * IMPORTANT: Any change here requires regenerating the index
 * with `pnpm run generate-llms`.
 */

const STOP_WORDS = new Set([
  // Articles & determiners
  "a",
  "an",
  "the",
  "this",
  "that",
  "these",
  "those",
  // Pronouns
  "i",
  "me",
  "my",
  "we",
  "our",
  "you",
  "your",
  "he",
  "she",
  "it",
  "its",
  "they",
  "them",
  "their",
  // Prepositions
  "in",
  "on",
  "at",
  "to",
  "of",
  "for",
  "from",
  "by",
  "with",
  "about",
  "into",
  "between",
  "through",
  "during",
  "before",
  "after",
  "above",
  "below",
  "up",
  "down",
  "out",
  "off",
  "over",
  "under",
  // Conjunctions
  "and",
  "but",
  "or",
  "nor",
  "so",
  "yet",
  // Verbs (common/auxiliary)
  "is",
  "am",
  "are",
  "was",
  "were",
  "be",
  "been",
  "being",
  "have",
  "has",
  "had",
  "do",
  "does",
  "did",
  "will",
  "would",
  "shall",
  "should",
  "may",
  "might",
  "must",
  "can",
  "could",
  // Question words (common in NL queries)
  "how",
  "what",
  "when",
  "where",
  "which",
  "who",
  "whom",
  "why",
  // Other common words
  "not",
  "no",
  "all",
  "each",
  "every",
  "both",
  "few",
  "more",
  "most",
  "other",
  "some",
  "such",
  "than",
  "too",
  "very",
  "just",
  "also",
  "here",
  "there",
  "then",
  "now",
]);

/**
 * MiniSearch processTerm — lowercases, filters stop words, and drops
 * empty tokens produced by the default tokenizer for trailing punctuation
 * (e.g. `hatchet.task(` → tokens: ["hatchet", "task", ""]).
 *
 * Must be identical at index time and query time.
 */
function processTerm(term: string): string | null {
  if (term.length === 0) return null;
  const lower = term.toLowerCase();
  if (STOP_WORDS.has(lower)) return null;
  return lower;
}

/**
 * MiniSearch options — must be passed identically to `new MiniSearch()`
 * and `MiniSearch.loadJSON()`.
 *
 * The `codeIdentifiers` field contains compound code tokens extracted from
 * fenced code blocks (e.g. "hatchet.task", "ctx.spawn"). These are indexed
 * as single tokens so that code-pattern queries match precisely.
 */
export const MINISEARCH_OPTIONS = {
  fields: ["title", "content", "codeIdentifiers"] as string[],
  storeFields: ["title", "pageTitle", "pageRoute"] as string[],
  processTerm,
};

/**
 * Default search options for querying the index.
 */
export const SEARCH_OPTIONS = {
  boost: { title: 2, codeIdentifiers: 3 },
  prefix: true,
  fuzzy: 0.2,
  combineWith: "OR" as const,
};

// ---------------------------------------------------------------------------
// Synonym / alias expansion
// ---------------------------------------------------------------------------

/**
 * Maps common synonyms, abbreviations, and alternate phrasings to terms
 * that actually appear in the documentation. This lets users find pages
 * even when they use different vocabulary than the docs.
 *
 * Keys are lowercased. Values are additional terms to append to the query.
 * The original query terms are always kept.
 */
const SYNONYMS: Record<string, string> = {
  // Scheduling & timing
  delay: "schedule sleep durable",
  pause: "sleep durable",
  debounce: "concurrency",
  dedup: "concurrency deduplicate idempotent",
  deduplicate: "concurrency idempotent",
  idempotent: "concurrency retry additional-metadata",
  recurring: "cron",
  periodic: "cron scheduled",
  interval: "cron scheduled",
  timer: "cron scheduled sleep",

  // Execution patterns
  "background job": "task run worker",
  "background task": "task run worker",
  enqueue: "run-no-wait",
  dispatch: "run trigger",
  invoke: "run trigger",
  trigger: "run event cron scheduled",
  "fan out": "child spawn bulk",
  fanout: "child spawn bulk",
  parallel: "child spawn run-with-results",
  scatter: "child spawn",
  gather: "child run-with-results",
  batch: "bulk run",
  "fire and forget": "run-no-wait",
  "long running": "durable execution",
  async: "asyncio",
  await: "asyncio run-with-results",

  // Error handling & reliability
  "error handling": "retry on-failure",
  "error recovery": "retry on-failure durable",
  fallback: "on-failure",
  "try catch": "retry on-failure",
  exception: "retry on-failure",
  resilience: "retry durable on-failure",
  reliable: "durable retry guarantees",

  // Observability & debugging
  monitor: "prometheus opentelemetry logging metrics",
  monitoring: "prometheus opentelemetry logging metrics",
  tracing: "opentelemetry",
  traces: "opentelemetry sampling",
  observability: "opentelemetry prometheus logging metrics",
  metrics: "prometheus opentelemetry",
  debug: "troubleshooting logging",
  troubleshoot: "troubleshooting workers",
  "not working": "troubleshooting workers",

  // Infrastructure & deployment
  deploy: "docker kubernetes compute",
  install: "setup quickstart",
  "getting started": "quickstart setup",
  "env var": "environment variable configuration",
  "env vars": "environment variables configuration",
  "environment variable": "configuration compute",
  scale: "autoscaling workers",
  autoscale: "autoscaling workers",
  "high availability": "ha helm",
  postgres: "database configuration external",
  database: "postgres configuration external",
  performance: "improving benchmarking",
  benchmark: "benchmarking performance",
  upgrade: "migration guide",
  downgrade: "downgrading versions",
  migrate: "migration guide",

  // Concurrency & flow control
  throttle: "rate limit concurrency",
  "rate limiting": "rate limits",
  limit: "rate concurrency",
  lock: "concurrency",
  semaphore: "concurrency",
  mutex: "concurrency",

  // Workflow patterns
  step: "dag task workflow",
  pipeline: "dag workflow orchestration",
  graph: "dag workflow",
  "if else": "conditional workflows",
  branch: "conditional workflows",
  condition: "conditional workflows",
  orchestrate: "orchestration dag workflow",

  // Worker & execution concepts
  queue: "task concurrency worker",
  "job queue": "task worker",
  "task queue": "task worker concurrency",
  slot: "manual release concurrency worker",
  "health check": "healthcheck worker",
  liveness: "healthcheck worker",
  readiness: "healthcheck worker",
  sticky: "sticky assignment worker affinity",
  affinity: "worker affinity sticky",

  // Communication & events
  signal: "event durable",
  callback: "event webhook",
  hook: "webhook event",
  "wait for event": "durable events",
  subscribe: "event trigger",
  publish: "event trigger",
  "api call": "inter-service triggering",

  // SDK specific
  decorator: "hatchet.task python",
  middleware: "lifecycle dependency-injection",
  context: "ctx spawn",
  dataclass: "dataclasses pydantic python",
  lifespan: "lifespans lifecycle worker",
  cleanup: "lifespans lifecycle",
  teardown: "lifespans lifecycle",
  startup: "lifespans lifecycle worker",
  "type safe": "pydantic dataclasses",

  // CLI & tools
  terminal: "cli tui",
  "command line": "cli",
  dashboard: "tui",
};

/**
 * Expand a search query by appending synonym terms.
 *
 * For each word (or consecutive word pair) in the query that matches
 * a synonym key, the mapped terms are appended. The original query
 * is always preserved so exact matches still work.
 */
export function expandSynonyms(query: string): string {
  const lower = query.toLowerCase().trim();
  const words = lower.split(/\s+/).filter((w) => w.length > 0);
  const extra: string[] = [];

  // Check full query first
  if (SYNONYMS[lower]) {
    extra.push(SYNONYMS[lower]);
  }

  // Check bigrams (consecutive word pairs)
  for (let i = 0; i < words.length - 1; i++) {
    const bigram = words[i] + " " + words[i + 1];
    if (SYNONYMS[bigram]) {
      extra.push(SYNONYMS[bigram]);
    }
  }

  // Check individual words
  for (const word of words) {
    if (SYNONYMS[word]) {
      extra.push(SYNONYMS[word]);
    }
  }

  if (extra.length === 0) return query;
  return query + " " + extra.join(" ");
}

/**
 * Tokenize a query string similarly to MiniSearch — split on common
 * punctuation and whitespace, then lowercase and filter stop words.
 * This mirrors the default tokenizer's behavior for reranking purposes.
 */
function tokenizeQuery(text: string): string[] {
  return text
    .split(/[\s\-_.,:;!?'"()[\]{}<>@#$%^&*+=|/\\~`]+/)
    .map((t) => t.toLowerCase())
    .filter((t) => t.length > 0 && !STOP_WORDS.has(t));
}

/**
 * Post-search reranking: boost results whose title or route closely matches
 * the query.
 *
 * BM25 scores terms independently, so a title that is a near-exact match
 * for the full query (e.g. "Durable Execution" for "durable execut") may
 * rank below documents that score well on individual terms. This fixes that.
 */
export function rerankResults<T extends { id: string; score: number; [k: string]: unknown }>(
  results: T[],
  query: string,
): T[] {
  const queryLower = query.toLowerCase().trim();
  const queryTerms = tokenizeQuery(queryLower);

  if (queryTerms.length === 0) return results;

  return results
    .map((r) => {
      const title = ((r.title as string) || "").toLowerCase();
      // Strip the hatchet://docs/ prefix so it doesn't pollute route matching
      const rawRoute = ((r.pageRoute as string) || r.id || "").toLowerCase();
      const route = rawRoute.replace("hatchet://docs/", "");
      let boost = 1;

      // Big boost if the full query (cleaned) appears in the title
      const queryClean = queryTerms.join(" ");
      if (title.includes(queryClean)) {
        boost *= 5;
      }

      // Boost for each query term found in the title
      let titleTermHits = 0;
      for (const term of queryTerms) {
        if (title.includes(term)) titleTermHits++;
      }
      if (queryTerms.length > 0) {
        boost *= 1 + (titleTermHits / queryTerms.length) * 2;
      }

      // Smaller boost for query terms found in the page route / slug
      // (e.g. "task" matches "your-first-task" in the URL)
      let routeTermHits = 0;
      for (const term of queryTerms) {
        if (route.includes(term)) routeTermHits++;
      }
      if (queryTerms.length > 0) {
        boost *= 1 + (routeTermHits / queryTerms.length) * 0.5;
      }

      return { ...r, score: r.score * boost };
    })
    .sort((a, b) => b.score - a.score);
}
