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
    expectAnyOf: ["essentials/your-first-task"],
  },
  {
    name: "hatchet.task — without parens",
    query: "hatchet.task",
    expectAnyOf: ["essentials/your-first-task"],
  },
  {
    name: "@hatchet.task() — Python decorator",
    query: "@hatchet.task()",
    expectAnyOf: ["essentials/your-first-task"],
  },
  {
    name: "hatchet.workflow — defining a workflow",
    query: "hatchet.workflow",
    expectAnyOf: ["concepts/durable-workflows", "concepts/tasks", "concepts/priority"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // Getting started & onboarding
  // -------------------------------------------------------------------------
  {
    name: "quickstart",
    query: "quickstart",
    expectAnyOf: ["essentials/quickstart", "self-hosting/kubernetes-quickstart"],
  },
  {
    name: "setup",
    query: "setup",
    expectAnyOf: ["essentials/advanced", "essentials/quickstart"],
    topN: 10,
  },
  {
    name: "getting started",
    query: "getting started",
    expectAnyOf: ["essentials/quickstart", "essentials/advanced"],
    topN: 10,
  },
  {
    name: "install",
    query: "install",
    expectAnyOf: ["essentials/quickstart", "essentials/advanced", "reference/cli"],
    topN: 10,
  },
  {
    name: "architecture",
    query: "architecture",
    expectAnyOf: ["essentials/architecture-and-guarantees"],
  },
  {
    name: "guarantees",
    query: "guarantees",
    expectAnyOf: ["essentials/architecture-and-guarantees"],
  },

  // -------------------------------------------------------------------------
  // Core task features
  // -------------------------------------------------------------------------
  {
    name: "define a task",
    query: "define a task",
    expectAnyOf: ["essentials/your-first-task"],
    topN: 10,
  },
  {
    name: "create worker",
    query: "create worker",
    expectAnyOf: ["essentials/workers"],
    topN: 10,
  },
  {
    name: "worker",
    query: "worker",
    expectAnyOf: ["essentials/workers", "concepts/workers"],
  },
  {
    name: "run task",
    query: "run task",
    expectAnyOf: ["essentials/running-your-task", "concepts/run-with-results"],
    topN: 10,
  },
  {
    name: "environments",
    query: "environments",
    expectAnyOf: ["essentials/advanced"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // Trigger types
  // -------------------------------------------------------------------------
  {
    name: "run with results",
    query: "run with results",
    expectAnyOf: ["concepts/run-with-results"],
  },
  {
    name: "run no wait",
    query: "run no wait",
    expectAnyOf: ["concepts/run-no-wait"],
  },
  {
    name: "scheduled runs",
    query: "scheduled runs",
    expectAnyOf: ["concepts/scheduled-runs"],
  },
  {
    name: "cron",
    query: "cron",
    expectAnyOf: ["concepts/cron-runs"],
  },
  {
    name: "event trigger",
    query: "event trigger",
    expectAnyOf: ["concepts/run-on-event"],
    topN: 10,
  },
  {
    name: "bulk run",
    query: "bulk run",
    expectAnyOf: ["concepts/bulk-run"],
  },
  {
    name: "webhooks",
    query: "webhooks",
    expectAnyOf: ["concepts/webhooks"],
  },
  {
    name: "inter-service",
    query: "inter-service",
    expectAnyOf: ["concepts/inter-service-triggering"],
  },

  // -------------------------------------------------------------------------
  // Flow control
  // -------------------------------------------------------------------------
  {
    name: "concurrency",
    query: "concurrency",
    expectAnyOf: ["concepts/concurrency"],
  },
  {
    name: "rate limit",
    query: "rate limit",
    expectAnyOf: ["concepts/rate-limits"],
  },
  {
    name: "rate limits (plural)",
    query: "rate limits",
    expectAnyOf: ["concepts/rate-limits"],
  },
  {
    name: "priority",
    query: "priority",
    expectAnyOf: ["concepts/priority"],
  },

  // -------------------------------------------------------------------------
  // Orchestration & composition
  // -------------------------------------------------------------------------
  {
    name: "orchestration",
    query: "orchestration",
    expectAnyOf: ["concepts/durable-workflows"],
    topN: 10,
  },
  {
    name: "DAG",
    query: "DAG",
    expectAnyOf: ["concepts/durable-workflows"],
    topN: 10,
  },
  {
    name: "conditional workflows",
    query: "conditional workflows",
    expectAnyOf: ["concepts/durable-workflows"],
    topN: 10,
  },
  {
    name: "on failure",
    query: "on failure",
    expectAnyOf: ["concepts/durable-workflows", "concepts/retry-policies"],
    topN: 10,
  },
  {
    name: "child spawning",
    query: "child spawning",
    expectAnyOf: ["concepts/durable-workflows", "concepts/run-with-results"],
    topN: 10,
  },
  {
    name: "child tasks",
    query: "child tasks",
    expectAnyOf: ["concepts/durable-workflows", "concepts/run-with-results", "concepts/sticky-assignment"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // Durability
  // -------------------------------------------------------------------------
  {
    name: "durable execution",
    query: "durable execution",
    expectAnyOf: ["concepts/durable-workflows", "essentials/intro-to-durable-workflows"],
  },
  {
    name: "durable events",
    query: "durable events",
    expectAnyOf: ["concepts/durable-workflows", "essentials/intro-to-durable-workflows"],
  },
  {
    name: "durable sleep",
    query: "durable sleep",
    expectAnyOf: ["concepts/durable-workflows", "essentials/intro-to-durable-workflows"],
  },
  {
    name: "durable best practices",
    query: "durable best practices",
    expectAnyOf: ["concepts/durable-workflows", "essentials/intro-to-durable-workflows"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // Reliability & error handling
  // -------------------------------------------------------------------------
  {
    name: "retry",
    query: "retry",
    expectAnyOf: ["concepts/retry-policies"],
  },
  {
    name: "timeout",
    query: "timeout",
    expectAnyOf: ["concepts/timeouts"],
  },
  {
    name: "cancellation",
    query: "cancellation",
    expectAnyOf: ["concepts/cancellation"],
  },
  {
    name: "bulk retries",
    query: "bulk retries",
    expectAnyOf: ["concepts/bulk-retries-and-cancellations"],
  },

  // -------------------------------------------------------------------------
  // Worker management
  // -------------------------------------------------------------------------
  {
    name: "sticky assignment",
    query: "sticky assignment",
    expectAnyOf: ["concepts/sticky-assignment"],
  },
  {
    name: "worker affinity",
    query: "worker affinity",
    expectAnyOf: ["concepts/worker-affinity"],
  },
  {
    name: "manual slot release",
    query: "manual slot release",
    expectAnyOf: ["concepts/manual-slot-release"],
  },
  {
    name: "autoscaling workers",
    query: "autoscaling workers",
    expectAnyOf: ["concepts/autoscaling-workers"],
  },
  {
    name: "worker health check",
    query: "worker health check",
    expectAnyOf: ["concepts/worker-healthchecks"],
    topN: 10,
  },
  {
    name: "troubleshooting",
    query: "troubleshooting",
    expectAnyOf: ["essentials/troubleshooting-workers"],
  },

  // -------------------------------------------------------------------------
  // Observability
  // -------------------------------------------------------------------------
  {
    name: "logging",
    query: "logging",
    expectAnyOf: ["concepts/logging"],
  },
  {
    name: "opentelemetry",
    query: "opentelemetry",
    expectAnyOf: ["concepts/opentelemetry"],
  },
  {
    name: "prometheus metrics",
    query: "prometheus metrics",
    expectAnyOf: ["self-hosting/prometheus-metrics", "concepts/prometheus-metrics"],
  },
  {
    name: "streaming",
    query: "streaming",
    expectAnyOf: ["concepts/streaming"],
  },
  {
    name: "additional metadata",
    query: "additional metadata",
    expectAnyOf: ["concepts/durable-workflows", "concepts/bulk-retries-and-cancellations"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // SDK-specific (Python)
  // -------------------------------------------------------------------------
  {
    name: "pydantic",
    query: "pydantic",
    expectAnyOf: ["reference/python/pydantic"],
    skip: true,
  },
  {
    name: "asyncio",
    query: "asyncio",
    expectAnyOf: ["reference/python/asyncio"],
    skip: true,
  },
  {
    name: "dependency injection",
    query: "dependency injection",
    expectAnyOf: ["reference/python/dependency-injection"],
    skip: true,
  },
  {
    name: "dataclass",
    query: "dataclass",
    expectAnyOf: ["reference/python/dataclasses"],
    skip: true,
  },
  {
    name: "lifespans",
    query: "lifespans",
    expectAnyOf: ["reference/python/lifespans"],
    skip: true,
  },

  // -------------------------------------------------------------------------
  // Migration guides
  // -------------------------------------------------------------------------
  {
    name: "migration python",
    query: "migration python",
    expectAnyOf: ["migrating/v0-to-v1/migration-guide-python"],
    skip: true,
  },
  {
    name: "migration typescript",
    query: "migration typescript",
    expectAnyOf: ["migrating/v0-to-v1/migration-guide-typescript"],
    skip: true,
  },
  {
    name: "migration go",
    query: "migration go",
    expectAnyOf: ["migrating/v0-to-v1/migration-guide-go"],
    skip: true,
  },
  {
    name: "engine migration",
    query: "engine migration",
    expectAnyOf: ["migrating/v0-to-v1/migration-guide-engine"],
    skip: true,
  },
  {
    name: "SDK improvements",
    query: "SDK improvements",
    expectAnyOf: ["migrating/v0-to-v1/v1-sdk-improvements"],
    skip: true,
  },

  // -------------------------------------------------------------------------
  // Self-hosting & infrastructure
  // -------------------------------------------------------------------------
  {
    name: "docker compose",
    query: "docker compose",
    expectAnyOf: ["self-hosting/docker-compose", "concepts/docker"],
  },
  {
    name: "running with docker",
    query: "running with docker",
    expectAnyOf: ["concepts/docker", "self-hosting/docker-compose"],
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
    expectAnyOf: ["reference/cli"],
  },
  {
    name: "TUI",
    query: "TUI",
    expectAnyOf: ["reference/cli"],
    topN: 10,
  },
  {
    name: "profiles",
    query: "profiles",
    expectAnyOf: ["reference/cli"],
    topN: 10,
  },
  {
    name: "running hatchet locally",
    query: "running hatchet locally",
    expectAnyOf: ["reference/cli", "self-hosting/hatchet-lite", "essentials/quickstart"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // Code-specific searches
  // -------------------------------------------------------------------------
  {
    name: "SimpleInput — Pydantic model",
    query: "SimpleInput",
    expectAnyOf: ["essentials/your-first-task"],
  },
  {
    name: "input_validator — Python arg",
    query: "input_validator",
    expectAnyOf: ["reference/python/pydantic", "essentials/your-first-task"],
  },
  {
    name: "BaseModel — Pydantic",
    query: "BaseModel",
    expectAnyOf: ["reference/python/pydantic", "essentials/your-first-task"],
  },
  {
    name: "ctx.spawn — child spawn",
    query: "ctx.spawn",
    expectAnyOf: ["concepts/durable-workflows", "concepts/run-with-results"],
    topN: 10,
  },
  {
    name: "NewStandaloneTask — Go API",
    query: "NewStandaloneTask",
    expectAnyOf: ["essentials/your-first-task", "migrating/v0-to-v1/migration-guide-go"],
  },
  {
    name: "DurableContext",
    query: "DurableContext",
    expectAnyOf: ["concepts/durable-workflows", "essentials/intro-to-durable-workflows"],
    skip: true,
  },
  {
    name: "aio_run — Python async run",
    query: "aio_run",
    expectAnyOf: ["essentials/your-first-task", "concepts/run-with-results"],
  },

  // -------------------------------------------------------------------------
  // Special characters (regression tests)
  // -------------------------------------------------------------------------
  {
    name: "hatchet.task( — trailing paren",
    query: "hatchet.task(",
    expectAnyOf: ["essentials/your-first-task"],
    topN: 10,
  },
  {
    name: "ctx.spawn( — trailing paren",
    query: "ctx.spawn(",
    expectAnyOf: ["concepts/durable-workflows", "concepts/run-with-results"],
    topN: 10,
  },
  {
    name: ".run() — dot prefix and parens",
    query: ".run()",
    expectAnyOf: ["essentials/your-first-task", "concepts/run-with-results", "essentials/running-your-task"],
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
    expectAnyOf: ["concepts/durable-workflows", "concepts/scheduled-runs"],
  },
  {
    name: "debounce → concurrency",
    query: "debounce",
    expectAnyOf: ["concepts/concurrency"],
  },
  {
    name: "dedup → concurrency",
    query: "dedup",
    expectAnyOf: ["concepts/concurrency"],
  },
  {
    name: "throttle → rate limits",
    query: "throttle",
    expectAnyOf: ["concepts/rate-limits", "concepts/concurrency"],
  },
  {
    name: "fan out → child spawning",
    query: "fan out",
    expectAnyOf: ["concepts/durable-workflows", "concepts/bulk-run", "patterns/fanout"],
  },
  {
    name: "parallel tasks",
    query: "parallel tasks",
    expectAnyOf: ["concepts/durable-workflows", "concepts/run-with-results"],
    topN: 10,
  },
  {
    name: "background job",
    query: "background job",
    expectAnyOf: ["essentials/your-first-task", "concepts/run-no-wait", "essentials/workers"],
  },
  {
    name: "recurring → cron",
    query: "recurring",
    expectAnyOf: ["concepts/cron-runs"],
  },
  {
    name: "error handling → retry/failure",
    query: "error handling",
    expectAnyOf: ["concepts/retry-policies", "concepts/durable-workflows"],
    topN: 10,
  },
  {
    name: "fire and forget → run no wait",
    query: "fire and forget",
    expectAnyOf: ["concepts/run-no-wait"],
    topN: 10,
  },
  {
    name: "scale workers → autoscaling",
    query: "scale workers",
    expectAnyOf: ["concepts/autoscaling-workers"],
  },
  {
    name: "pipeline → DAG",
    query: "pipeline",
    expectAnyOf: ["concepts/durable-workflows", "patterns/pre-determined-pipelines"],
  },
  {
    name: "long running task → durable",
    query: "long running task",
    expectAnyOf: ["concepts/durable-workflows", "essentials/intro-to-durable-workflows"],
    topN: 10,
  },
  {
    name: "batch → bulk run",
    query: "batch tasks",
    expectAnyOf: ["concepts/bulk-run", "patterns/batch-processing"],
    topN: 10,
  },
  {
    name: "if else → conditional",
    query: "if else workflow",
    expectAnyOf: ["concepts/durable-workflows", "patterns/branching"],
    topN: 10,
  },
  {
    name: "monitor → observability",
    query: "monitor",
    expectAnyOf: ["concepts/opentelemetry", "concepts/prometheus-metrics", "concepts/logging"],
    topN: 10,
  },
  {
    name: "tracing → opentelemetry",
    query: "tracing",
    expectAnyOf: ["concepts/opentelemetry"],
    topN: 10,
  },
  {
    name: "observability",
    query: "observability",
    expectAnyOf: ["concepts/opentelemetry", "concepts/prometheus-metrics", "concepts/logging"],
    topN: 10,
  },
  {
    name: "debug → troubleshooting",
    query: "debug",
    expectAnyOf: ["essentials/troubleshooting-workers", "concepts/logging"],
    topN: 10,
  },
  {
    name: "deploy → docker/k8s",
    query: "deploy",
    expectAnyOf: ["concepts/docker", "self-hosting/docker-compose", "self-hosting/kubernetes-quickstart"],
    topN: 10,
  },
  {
    name: "upgrade → migration",
    query: "upgrade",
    expectAnyOf: ["migrating/v0-to-v1/migration-guide-python", "migrating/v0-to-v1/migration-guide-typescript", "migrating/v0-to-v1/migration-guide-go", "migrating/v0-to-v1/migration-guide-engine", "self-hosting/upgrading-downgrading"],
    topN: 10,
  },
  {
    name: "downgrade → downgrading",
    query: "downgrade",
    expectAnyOf: ["self-hosting/downgrading-db-schema-manually", "self-hosting/upgrading-downgrading"],
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
    expectAnyOf: ["reference/python/asyncio"],
    topN: 10,
    skip: true,
  },
  {
    name: "liveness → health checks",
    query: "liveness",
    expectAnyOf: ["concepts/worker-healthchecks"],
    topN: 10,
  },
  {
    name: "wait for event → durable events",
    query: "wait for event",
    expectAnyOf: ["concepts/durable-workflows", "essentials/intro-to-durable-workflows", "concepts/pushing-events", "concepts/event-filters", "patterns/long-waits", "patterns/event-driven"],
    topN: 10,
  },
  {
    name: "api call → inter-service",
    query: "api call between services",
    expectAnyOf: ["concepts/inter-service-triggering"],
    topN: 10,
  },
  {
    name: "cleanup → lifespans",
    query: "cleanup shutdown",
    expectAnyOf: ["reference/python/lifespans"],
    topN: 10,
    skip: true,
  },

  // -------------------------------------------------------------------------
  // Natural language questions
  // -------------------------------------------------------------------------
  {
    name: "how to retry a failed task",
    query: "how to retry a failed task",
    expectAnyOf: ["concepts/retry-policies", "concepts/durable-workflows"],
    topN: 10,
  },
  {
    name: "how to run tasks in parallel",
    query: "how to run tasks in parallel",
    expectAnyOf: ["concepts/durable-workflows", "concepts/run-with-results"],
    topN: 10,
  },
  {
    name: "how to cancel a running task",
    query: "how to cancel a running task",
    expectAnyOf: ["concepts/cancellation"],
    topN: 10,
  },
  {
    name: "how to set up cron job",
    query: "how to set up cron job",
    expectAnyOf: ["concepts/cron-runs"],
    topN: 10,
  },
  {
    name: "how to handle errors",
    query: "how to handle errors",
    expectAnyOf: ["concepts/retry-policies", "concepts/durable-workflows"],
    topN: 10,
  },
  {
    name: "how to limit concurrency",
    query: "how to limit concurrency",
    expectAnyOf: ["concepts/concurrency", "concepts/rate-limits"],
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
