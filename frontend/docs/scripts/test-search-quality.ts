/**
 * Search quality test harness for the docs MiniSearch index.
 *
 * Defines a set of common search queries with expected results,
 * runs them against the generated index, and reports pass/fail.
 *
 * Usage:
 *   tsx scripts/test-search-quality.ts
 *
 * Exit code 0 = all tests pass, 1 = failures detected.
 */

import fs from "node:fs";
import path from "node:path";
import MiniSearch from "minisearch";
import { MINISEARCH_OPTIONS, SEARCH_OPTIONS, rerankResults, expandSynonyms } from "../lib/search-config.js";

// ---------------------------------------------------------------------------
// Load the search index
// ---------------------------------------------------------------------------
const SCRIPT_DIR = path.dirname(new URL(import.meta.url).pathname);
const DOCS_ROOT = path.resolve(SCRIPT_DIR, "..");
const INDEX_PATH = path.join(DOCS_ROOT, "public", "llms-search-index.json");

function loadIndex(): MiniSearch {
  const json = fs.readFileSync(INDEX_PATH, "utf-8");
  return MiniSearch.loadJSON(json, MINISEARCH_OPTIONS);
}

// ---------------------------------------------------------------------------
// Test case definitions
// ---------------------------------------------------------------------------

interface SearchTestCase {
  /** Human description of what we're testing */
  name: string;
  /** The raw search query (exactly what a user would type) */
  query: string;
  /** At least one of these page routes must appear in the top N results */
  expectAnyOf: string[];
  /** How many top results to check (default: 5) */
  topN?: number;
  /** If true, skip this test (for known issues / WIP) */
  skip?: boolean;
}

const TEST_CASES: SearchTestCase[] = [
  // -------------------------------------------------------------------------
  // Core API patterns — things developers commonly search for
  // -------------------------------------------------------------------------
  {
    name: "hatchet.task( — defining a task",
    query: "hatchet.task(",
    expectAnyOf: ["home/your-first-task"],
  },
  {
    name: "hatchet.task — without parens",
    query: "hatchet.task",
    expectAnyOf: ["home/your-first-task"],
  },
  {
    name: "@hatchet.task() — Python decorator",
    query: "@hatchet.task()",
    expectAnyOf: ["home/your-first-task"],
  },
  {
    name: "hatchet.workflow — defining a workflow",
    query: "hatchet.workflow",
    expectAnyOf: ["home/dags", "home/orchestration"],
  },

  // -------------------------------------------------------------------------
  // Getting started & onboarding
  // -------------------------------------------------------------------------
  {
    name: "quickstart",
    query: "quickstart",
    expectAnyOf: ["home/hatchet-cloud-quickstart", "self-hosting/kubernetes-quickstart"],
  },
  {
    name: "setup",
    query: "setup",
    expectAnyOf: ["home/setup", "home/hatchet-cloud-quickstart"],
  },
  {
    name: "getting started",
    query: "getting started",
    expectAnyOf: ["home/hatchet-cloud-quickstart", "home/setup"],
    topN: 10,
  },
  {
    name: "install",
    query: "install",
    expectAnyOf: ["home/hatchet-cloud-quickstart", "home/setup", "cli/index"],
    topN: 10,
  },
  {
    name: "architecture",
    query: "architecture",
    expectAnyOf: ["home/architecture"],
  },
  {
    name: "guarantees",
    query: "guarantees",
    expectAnyOf: ["home/guarantees-and-tradeoffs"],
  },

  // -------------------------------------------------------------------------
  // Core task features
  // -------------------------------------------------------------------------
  {
    name: "define a task",
    query: "define a task",
    expectAnyOf: ["home/your-first-task"],
    topN: 10,
  },
  {
    name: "create worker",
    query: "create worker",
    expectAnyOf: ["home/workers"],
    topN: 10,
  },
  {
    name: "worker",
    query: "worker",
    expectAnyOf: ["home/workers"],
  },
  {
    name: "run task",
    query: "run task",
    expectAnyOf: ["home/running-your-task", "home/running-tasks", "home/run-with-results"],
    topN: 10,
  },
  {
    name: "environments",
    query: "environments",
    expectAnyOf: ["home/environments"],
  },

  // -------------------------------------------------------------------------
  // Trigger types
  // -------------------------------------------------------------------------
  {
    name: "run with results",
    query: "run with results",
    expectAnyOf: ["home/run-with-results"],
  },
  {
    name: "run no wait",
    query: "run no wait",
    expectAnyOf: ["home/run-no-wait"],
  },
  {
    name: "scheduled runs",
    query: "scheduled runs",
    expectAnyOf: ["home/scheduled-runs"],
  },
  {
    name: "cron",
    query: "cron",
    expectAnyOf: ["home/cron-runs"],
  },
  {
    name: "event trigger",
    query: "event trigger",
    expectAnyOf: ["home/run-on-event"],
    topN: 10,
  },
  {
    name: "bulk run",
    query: "bulk run",
    expectAnyOf: ["home/bulk-run"],
  },
  {
    name: "webhooks",
    query: "webhooks",
    expectAnyOf: ["home/webhooks"],
  },
  {
    name: "inter-service",
    query: "inter-service",
    expectAnyOf: ["home/inter-service-triggering"],
  },

  // -------------------------------------------------------------------------
  // Flow control
  // -------------------------------------------------------------------------
  {
    name: "concurrency",
    query: "concurrency",
    expectAnyOf: ["home/concurrency"],
  },
  {
    name: "rate limit",
    query: "rate limit",
    expectAnyOf: ["home/rate-limits"],
  },
  {
    name: "rate limits (plural)",
    query: "rate limits",
    expectAnyOf: ["home/rate-limits"],
  },
  {
    name: "priority",
    query: "priority",
    expectAnyOf: ["home/priority"],
  },

  // -------------------------------------------------------------------------
  // Orchestration & composition
  // -------------------------------------------------------------------------
  {
    name: "orchestration",
    query: "orchestration",
    expectAnyOf: ["home/orchestration"],
  },
  {
    name: "DAG",
    query: "DAG",
    expectAnyOf: ["home/dags"],
  },
  {
    name: "conditional workflows",
    query: "conditional workflows",
    expectAnyOf: ["home/conditional-workflows"],
  },
  {
    name: "on failure",
    query: "on failure",
    expectAnyOf: ["home/on-failure-tasks"],
  },
  {
    name: "child spawning",
    query: "child spawning",
    expectAnyOf: ["home/child-spawning"],
  },
  {
    name: "child tasks",
    query: "child tasks",
    expectAnyOf: ["home/child-spawning"],
  },

  // -------------------------------------------------------------------------
  // Durability
  // -------------------------------------------------------------------------
  {
    name: "durable execution",
    query: "durable execution",
    expectAnyOf: ["home/durable-execution"],
  },
  {
    name: "durable events",
    query: "durable events",
    expectAnyOf: ["home/durable-events"],
  },
  {
    name: "durable sleep",
    query: "durable sleep",
    expectAnyOf: ["home/durable-sleep"],
  },
  {
    name: "durable best practices",
    query: "durable best practices",
    expectAnyOf: ["home/durable-best-practices"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // Reliability & error handling
  // -------------------------------------------------------------------------
  {
    name: "retry",
    query: "retry",
    expectAnyOf: ["home/retry-policies"],
  },
  {
    name: "timeout",
    query: "timeout",
    expectAnyOf: ["home/timeouts"],
  },
  {
    name: "cancellation",
    query: "cancellation",
    expectAnyOf: ["home/cancellation"],
  },
  {
    name: "bulk retries",
    query: "bulk retries",
    expectAnyOf: ["home/bulk-retries-and-cancellations"],
  },

  // -------------------------------------------------------------------------
  // Worker management
  // -------------------------------------------------------------------------
  {
    name: "sticky assignment",
    query: "sticky assignment",
    expectAnyOf: ["home/sticky-assignment"],
  },
  {
    name: "worker affinity",
    query: "worker affinity",
    expectAnyOf: ["home/worker-affinity"],
  },
  {
    name: "manual slot release",
    query: "manual slot release",
    expectAnyOf: ["home/manual-slot-release"],
  },
  {
    name: "autoscaling workers",
    query: "autoscaling workers",
    expectAnyOf: ["home/autoscaling-workers"],
  },
  {
    name: "worker health check",
    query: "worker health check",
    expectAnyOf: ["home/worker-healthchecks"],
    topN: 10,
  },
  {
    name: "troubleshooting",
    query: "troubleshooting",
    expectAnyOf: ["home/troubleshooting-workers"],
  },

  // -------------------------------------------------------------------------
  // Observability
  // -------------------------------------------------------------------------
  {
    name: "logging",
    query: "logging",
    expectAnyOf: ["home/logging"],
  },
  {
    name: "opentelemetry",
    query: "opentelemetry",
    expectAnyOf: ["home/opentelemetry"],
  },
  {
    name: "prometheus metrics",
    query: "prometheus metrics",
    expectAnyOf: ["self-hosting/prometheus-metrics", "home/prometheus-metrics"],
  },
  {
    name: "streaming",
    query: "streaming",
    expectAnyOf: ["home/streaming"],
  },
  {
    name: "additional metadata",
    query: "additional metadata",
    expectAnyOf: ["home/additional-metadata"],
  },

  // -------------------------------------------------------------------------
  // SDK-specific (Python)
  // -------------------------------------------------------------------------
  {
    name: "pydantic",
    query: "pydantic",
    expectAnyOf: ["home/pydantic"],
  },
  {
    name: "asyncio",
    query: "asyncio",
    expectAnyOf: ["home/asyncio"],
  },
  {
    name: "dependency injection",
    query: "dependency injection",
    expectAnyOf: ["home/middleware"],
  },
  {
    name: "dataclass",
    query: "dataclass",
    expectAnyOf: ["home/dataclasses"],
  },
  {
    name: "lifespans",
    query: "lifespans",
    expectAnyOf: ["home/lifespans"],
  },

  // -------------------------------------------------------------------------
  // Migration guides
  // -------------------------------------------------------------------------
  {
    name: "migration python",
    query: "migration python",
    expectAnyOf: ["home/migration-guide-python"],
  },
  {
    name: "migration typescript",
    query: "migration typescript",
    expectAnyOf: ["home/migration-guide-typescript"],
  },
  {
    name: "migration go",
    query: "migration go",
    expectAnyOf: ["home/migration-guide-go"],
  },
  {
    name: "engine migration",
    query: "engine migration",
    expectAnyOf: ["home/migration-guide-engine"],
  },
  {
    name: "SDK improvements",
    query: "SDK improvements",
    expectAnyOf: ["home/v1-sdk-improvements"],
  },

  // -------------------------------------------------------------------------
  // Self-hosting & infrastructure
  // -------------------------------------------------------------------------
  {
    name: "docker compose",
    query: "docker compose",
    expectAnyOf: ["self-hosting/docker-compose", "home/docker"],
  },
  {
    name: "running with docker",
    query: "running with docker",
    expectAnyOf: ["home/docker", "self-hosting/docker-compose"],
    topN: 10,
  },
  {
    name: "kubernetes",
    query: "kubernetes",
    expectAnyOf: ["self-hosting/kubernetes-quickstart", "self-hosting/kubernetes-helm-configuration"],
  },
  {
    name: "helm chart",
    query: "helm chart",
    expectAnyOf: ["self-hosting/kubernetes-helm-configuration", "self-hosting/high-availability"],
  },
  {
    name: "configuration options",
    query: "configuration options",
    expectAnyOf: ["self-hosting/configuration-options"],
  },
  {
    name: "self hosting",
    query: "self hosting",
    expectAnyOf: ["self-hosting/index", "self-hosting/docker-compose"],
    topN: 10,
  },
  {
    name: "hatchet lite",
    query: "hatchet lite",
    expectAnyOf: ["self-hosting/hatchet-lite"],
  },
  {
    name: "networking",
    query: "networking",
    expectAnyOf: ["self-hosting/networking"],
  },
  {
    name: "external database",
    query: "external database",
    expectAnyOf: ["self-hosting/kubernetes-external-database"],
  },
  {
    name: "high availability",
    query: "high availability",
    expectAnyOf: ["self-hosting/high-availability"],
  },
  {
    name: "data retention",
    query: "data retention",
    expectAnyOf: ["self-hosting/data-retention"],
  },
  {
    name: "benchmarking",
    query: "benchmarking",
    expectAnyOf: ["self-hosting/benchmarking"],
  },
  {
    name: "read replicas",
    query: "read replicas",
    expectAnyOf: ["self-hosting/read-replicas"],
  },
  {
    name: "SMTP",
    query: "SMTP",
    expectAnyOf: ["self-hosting/smtp-server"],
  },
  {
    name: "sampling",
    query: "sampling",
    expectAnyOf: ["self-hosting/sampling"],
  },
  {
    name: "glasskube",
    query: "glasskube",
    expectAnyOf: ["self-hosting/kubernetes-glasskube"],
  },
  {
    name: "downgrading versions",
    query: "downgrading versions",
    expectAnyOf: ["self-hosting/downgrading-db-schema-manually"],
  },
  {
    name: "improving performance",
    query: "improving performance",
    expectAnyOf: ["self-hosting/improving-performance"],
  },
  {
    name: "worker configuration",
    query: "worker configuration",
    expectAnyOf: ["self-hosting/worker-configuration-options"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // CLI
  // -------------------------------------------------------------------------
  {
    name: "CLI",
    query: "CLI",
    expectAnyOf: ["cli/index"],
  },
  {
    name: "TUI",
    query: "TUI",
    expectAnyOf: ["cli/tui"],
  },
  {
    name: "profiles",
    query: "profiles",
    expectAnyOf: ["cli/profiles"],
  },
  {
    name: "running hatchet locally",
    query: "running hatchet locally",
    expectAnyOf: ["cli/running-hatchet-locally"],
  },

  // -------------------------------------------------------------------------
  // Code-specific searches
  // -------------------------------------------------------------------------
  {
    name: "SimpleInput — Pydantic model",
    query: "SimpleInput",
    expectAnyOf: ["home/your-first-task"],
  },
  {
    name: "input_validator — Python arg",
    query: "input_validator",
    expectAnyOf: ["home/pydantic", "home/your-first-task"],
  },
  {
    name: "BaseModel — Pydantic",
    query: "BaseModel",
    expectAnyOf: ["home/pydantic", "home/your-first-task"],
  },
  {
    name: "ctx.spawn — child spawn",
    query: "ctx.spawn",
    expectAnyOf: ["home/child-spawning"],
  },
  {
    name: "NewStandaloneTask — Go API",
    query: "NewStandaloneTask",
    expectAnyOf: ["home/your-first-task", "home/migration-guide-go"],
  },
  {
    name: "DurableContext",
    query: "DurableContext",
    expectAnyOf: ["home/durable-execution"],
  },
  {
    name: "aio_run — Python async run",
    query: "aio_run",
    expectAnyOf: ["home/your-first-task", "home/run-with-results"],
  },

  // -------------------------------------------------------------------------
  // Special characters (regression tests)
  // -------------------------------------------------------------------------
  {
    name: "hatchet.task( — trailing paren",
    query: "hatchet.task(",
    expectAnyOf: ["home/your-first-task"],
    topN: 10,
  },
  {
    name: "ctx.spawn( — trailing paren",
    query: "ctx.spawn(",
    expectAnyOf: ["home/child-spawning"],
    topN: 10,
  },
  {
    name: ".run() — dot prefix and parens",
    query: ".run()",
    expectAnyOf: ["home/your-first-task", "home/run-with-results", "home/running-your-task"],
    topN: 10,
  },
  {
    name: "( — lone paren should not crash",
    query: "(",
    expectAnyOf: [],
  },
  {
    name: ") — lone close paren should not crash",
    query: ")",
    expectAnyOf: [],
  },

  // -------------------------------------------------------------------------
  // Synonym / alternate phrasing queries
  // -------------------------------------------------------------------------
  {
    name: "delay → scheduled/sleep",
    query: "delay",
    expectAnyOf: ["home/durable-sleep", "home/scheduled-runs"],
  },
  {
    name: "debounce → concurrency",
    query: "debounce",
    expectAnyOf: ["home/concurrency"],
  },
  {
    name: "dedup → concurrency",
    query: "dedup",
    expectAnyOf: ["home/concurrency"],
  },
  {
    name: "throttle → rate limits",
    query: "throttle",
    expectAnyOf: ["home/rate-limits", "home/concurrency"],
  },
  {
    name: "fan out → child spawning",
    query: "fan out",
    expectAnyOf: ["home/child-spawning", "home/bulk-run"],
  },
  {
    name: "parallel tasks",
    query: "parallel tasks",
    expectAnyOf: ["home/child-spawning", "home/run-with-results"],
  },
  {
    name: "background job",
    query: "background job",
    expectAnyOf: ["home/your-first-task", "home/run-no-wait", "home/workers"],
  },
  {
    name: "recurring → cron",
    query: "recurring",
    expectAnyOf: ["home/cron-runs"],
  },
  {
    name: "error handling → retry/failure",
    query: "error handling",
    expectAnyOf: ["home/retry-policies", "home/on-failure-tasks"],
  },
  {
    name: "fire and forget → run no wait",
    query: "fire and forget",
    expectAnyOf: ["home/run-no-wait"],
    topN: 10,
  },
  {
    name: "scale workers → autoscaling",
    query: "scale workers",
    expectAnyOf: ["home/autoscaling-workers"],
  },
  {
    name: "pipeline → DAG",
    query: "pipeline",
    expectAnyOf: ["home/dags", "home/orchestration"],
  },
  {
    name: "long running task → durable",
    query: "long running task",
    expectAnyOf: ["home/durable-execution"],
    topN: 10,
  },
  {
    name: "batch → bulk run",
    query: "batch tasks",
    expectAnyOf: ["home/bulk-run"],
    topN: 10,
  },
  {
    name: "if else → conditional",
    query: "if else workflow",
    expectAnyOf: ["home/conditional-workflows"],
    topN: 10,
  },
  {
    name: "monitor → observability",
    query: "monitor",
    expectAnyOf: ["home/opentelemetry", "home/prometheus-metrics", "home/logging"],
    topN: 10,
  },
  {
    name: "tracing → opentelemetry",
    query: "tracing",
    expectAnyOf: ["home/opentelemetry"],
    topN: 10,
  },
  {
    name: "observability",
    query: "observability",
    expectAnyOf: ["home/opentelemetry", "home/prometheus-metrics", "home/logging"],
    topN: 10,
  },
  {
    name: "debug → troubleshooting",
    query: "debug",
    expectAnyOf: ["home/troubleshooting-workers", "home/logging"],
    topN: 10,
  },
  {
    name: "deploy → docker/k8s",
    query: "deploy",
    expectAnyOf: ["home/docker", "self-hosting/docker-compose", "self-hosting/kubernetes-quickstart"],
    topN: 10,
  },
  {
    name: "upgrade → migration",
    query: "upgrade",
    expectAnyOf: ["home/migration-guide-python", "home/migration-guide-typescript", "home/migration-guide-go", "home/migration-guide-engine"],
    topN: 10,
  },
  {
    name: "downgrade → downgrading",
    query: "downgrade",
    expectAnyOf: ["self-hosting/downgrading-db-schema-manually"],
    topN: 10,
  },
  {
    name: "postgres → database config",
    query: "postgres",
    expectAnyOf: ["self-hosting/kubernetes-external-database", "self-hosting/configuration-options"],
    topN: 10,
  },
  {
    name: "performance → improving",
    query: "performance",
    expectAnyOf: ["self-hosting/improving-performance", "self-hosting/benchmarking"],
    topN: 10,
  },
  {
    name: "async await → asyncio",
    query: "async await",
    expectAnyOf: ["home/asyncio"],
    topN: 10,
  },
  {
    name: "liveness → health checks",
    query: "liveness",
    expectAnyOf: ["home/worker-healthchecks"],
    topN: 10,
  },
  {
    name: "wait for event → durable events",
    query: "wait for event",
    expectAnyOf: ["home/durable-events"],
    topN: 10,
  },
  {
    name: "api call → inter-service",
    query: "api call between services",
    expectAnyOf: ["home/inter-service-triggering"],
    topN: 10,
  },
  {
    name: "cleanup → lifespans",
    query: "cleanup shutdown",
    expectAnyOf: ["home/lifespans"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // Natural language questions
  // -------------------------------------------------------------------------
  {
    name: "how to retry a failed task",
    query: "how to retry a failed task",
    expectAnyOf: ["home/retry-policies", "home/on-failure-tasks"],
    topN: 10,
  },
  {
    name: "how to run tasks in parallel",
    query: "how to run tasks in parallel",
    expectAnyOf: ["home/child-spawning", "home/run-with-results"],
    topN: 10,
  },
  {
    name: "how to cancel a running task",
    query: "how to cancel a running task",
    expectAnyOf: ["home/cancellation"],
    topN: 10,
  },
  {
    name: "how to set up cron job",
    query: "how to set up cron job",
    expectAnyOf: ["home/cron-runs"],
    topN: 10,
  },
  {
    name: "how to handle errors",
    query: "how to handle errors",
    expectAnyOf: ["home/retry-policies", "home/on-failure-tasks"],
    topN: 10,
  },
  {
    name: "how to limit concurrency",
    query: "how to limit concurrency",
    expectAnyOf: ["home/concurrency", "home/rate-limits"],
    topN: 10,
  },
];

// ---------------------------------------------------------------------------
// Test runner
// ---------------------------------------------------------------------------

interface TestResult {
  name: string;
  query: string;
  passed: boolean;
  reason?: string;
  topResults: Array<{ title: string; route: string; score: number }>;
}

function runTests(idx: MiniSearch): TestResult[] {
  const results: TestResult[] = [];

  for (const tc of TEST_CASES) {
    if (tc.skip) {
      results.push({
        name: tc.name,
        query: tc.query,
        passed: true,
        reason: "SKIPPED",
        topResults: [],
      });
      continue;
    }

    const topN = tc.topN ?? 5;

    let searchResults: any[];
    try {
      const expanded = expandSynonyms(tc.query);
      const raw = idx.search(expanded, SEARCH_OPTIONS);
      searchResults = rerankResults(raw, tc.query);
    } catch (e: any) {
      results.push({
        name: tc.name,
        query: tc.query,
        passed: false,
        reason: `Search threw: ${e.message}`,
        topResults: [],
      });
      continue;
    }

    // If no expected results, just check it didn't crash
    if (tc.expectAnyOf.length === 0) {
      results.push({
        name: tc.name,
        query: tc.query,
        passed: true,
        reason: "No crash (no expected results)",
        topResults: [],
      });
      continue;
    }

    const topSlice = searchResults.slice(0, topN);
    const topRoutes = topSlice.map((r) => {
      const route = (r.pageRoute as string || r.id).replace("hatchet://docs/", "");
      return route;
    });
    const topIds = topSlice.map((r) => r.id.replace("hatchet://docs/", ""));

    // Check if any expected route appears in top results (match on page route or section id)
    const found = tc.expectAnyOf.some(
      (expected) =>
        topRoutes.some((r) => r === expected || r.startsWith(expected + "#")) ||
        topIds.some((id) => id === expected || id.startsWith(expected + "#") || expected.includes("#") && id === expected),
    );

    results.push({
      name: tc.name,
      query: tc.query,
      passed: found,
      reason: found
        ? undefined
        : `Expected one of [${tc.expectAnyOf.join(", ")}] in top ${topN}, got: [${topIds.slice(0, topN).join(", ")}]`,
      topResults: topSlice.map((r) => ({
        title: r.title as string,
        route: r.id.replace("hatchet://docs/", ""),
        score: r.score,
      })),
    });
  }

  return results;
}

// ---------------------------------------------------------------------------
// Output formatting
// ---------------------------------------------------------------------------

function formatResults(results: TestResult[], warnMode: boolean): void {
  const passed = results.filter((r) => r.passed);
  const failed = results.filter((r) => !r.passed);

  if (warnMode) {
    // Compact output: only show failures as warnings
    if (failed.length === 0) {
      console.log(
        `  Search quality: ${passed.length}/${results.length} tests passed`,
      );
    } else {
      console.warn(
        `\n  ⚠ Search quality: ${failed.length}/${results.length} tests FAILED:`,
      );
      for (const r of failed) {
        console.warn(`    • ${r.name} (query: ${JSON.stringify(r.query)})`);
      }
      console.warn();
    }
    return;
  }

  console.log("╔════════════════════════════════════════════════════════════╗");
  console.log("║          Search Quality Test Results                      ║");
  console.log("╚════════════════════════════════════════════════════════════╝\n");

  if (failed.length > 0) {
    console.log(`❌ FAILURES (${failed.length}):\n`);
    for (const r of failed) {
      console.log(`  FAIL: ${r.name}`);
      console.log(`        query: ${JSON.stringify(r.query)}`);
      console.log(`        ${r.reason}`);
      if (r.topResults.length > 0) {
        console.log(`        actual top results:`);
        r.topResults.slice(0, 5).forEach((tr, i) => {
          console.log(`          ${i + 1}. [${tr.score.toFixed(1)}] ${tr.title} (${tr.route})`);
        });
      }
      console.log();
    }
  }

  if (passed.length > 0) {
    console.log(`✅ PASSED (${passed.length}):\n`);
    for (const r of passed) {
      const note = r.reason ? ` (${r.reason})` : "";
      console.log(`  OK: ${r.name}${note}`);
    }
    console.log();
  }

  console.log("─".repeat(60));
  console.log(
    `Total: ${results.length} | Passed: ${passed.length} | Failed: ${failed.length}`,
  );
  console.log("─".repeat(60));
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

function main(): void {
  const warnMode = process.argv.includes("--warn");

  if (!fs.existsSync(INDEX_PATH)) {
    if (warnMode) {
      console.warn("  ⚠ Search index not found — skipping search quality tests");
      process.exit(0);
    }
    console.error(
      "Search index not found. Run 'pnpm run generate-llms' first.",
    );
    process.exit(1);
  }

  if (!warnMode) {
    console.log("Loading search index...");
  }
  const idx = loadIndex();

  if (!warnMode) {
    console.log(`Running ${TEST_CASES.length} search quality tests...\n`);
  }
  const results = runTests(idx);

  formatResults(results, warnMode);

  const failed = results.filter((r) => !r.passed);
  // In --warn mode, always exit 0 so we don't block the dev server
  process.exit(warnMode ? 0 : failed.length > 0 ? 1 : 0);
}

main();
