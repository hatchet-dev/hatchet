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
    expectAnyOf: ["v1/tasks"],
  },
  {
    name: "hatchet.task — without parens",
    query: "hatchet.task",
    expectAnyOf: ["v1/tasks"],
  },
  {
    name: "@hatchet.task() — Python decorator",
    query: "@hatchet.task()",
    expectAnyOf: ["v1/tasks"],
  },
  {
    name: "hatchet.workflow — defining a workflow",
    query: "hatchet.workflow",
    expectAnyOf: ["v1/durable-execution", "v1/patterns/directed-acyclic-graphs", "v1/priority", "reference/python/runnables"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // Getting started & onboarding
  // -------------------------------------------------------------------------
  {
    name: "v1/quickstart",
    query: "v1/quickstart",
    expectAnyOf: ["v1/quickstart", "self-hosting/kubernetes-quickstart"],
  },
  {
    name: "setup",
    query: "setup",
    expectAnyOf: ["v1/quickstart", "reference/cli/index", "reference/cli"],
    topN: 10,
  },
  {
    name: "getting started",
    query: "getting started",
    expectAnyOf: ["v1/quickstart"],
    topN: 10,
  },
  {
    name: "install",
    query: "install",
    expectAnyOf: ["v1/quickstart", "reference/cli/index", "reference/cli"],
    topN: 10,
  },
  {
    name: "architecture",
    query: "architecture",
    expectAnyOf: ["v1/architecture-and-guarantees"],
  },
  {
    name: "guarantees",
    query: "guarantees",
    expectAnyOf: ["v1/architecture-and-guarantees"],
  },

  // -------------------------------------------------------------------------
  // Core task features
  // -------------------------------------------------------------------------
  {
    name: "define a task",
    query: "define a task",
    expectAnyOf: ["v1/tasks"],
    topN: 10,
  },
  {
    name: "create worker",
    query: "create worker",
    expectAnyOf: ["v1/workers"],
    topN: 10,
  },
  {
    name: "worker",
    query: "worker",
    expectAnyOf: ["v1/workers", "v1/runtime/workers"],
  },
  {
    name: "run task",
    query: "run task",
    expectAnyOf: ["v1/running-your-task"],
    topN: 10,
  },
  {
    name: "v1/environments",
    query: "v1/environments",
    expectAnyOf: ["v1/environments"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // Trigger types
  // -------------------------------------------------------------------------
  {
    name: "run with results",
    query: "run with results",
    expectAnyOf: ["v1/running-your-task"],
  },
  {
    name: "run no wait",
    query: "run no wait",
    expectAnyOf: ["v1/running-your-task"],
  },
  {
    name: "scheduled runs",
    query: "scheduled runs",
    expectAnyOf: ["v1/scheduled-runs"],
  },
  {
    name: "cron",
    query: "cron",
    expectAnyOf: ["v1/cron-runs"],
  },
  {
    name: "event trigger",
    query: "event trigger",
    expectAnyOf: ["v1/external-events/run-on-event"],
    topN: 10,
  },
  {
    name: "bulk run",
    query: "bulk run",
    expectAnyOf: ["v1/bulk-run"],
  },
  {
    name: "webhooks",
    query: "webhooks",
    expectAnyOf: ["v1/webhooks"],
  },
  {
    name: "inter-service",
    query: "inter-service",
    expectAnyOf: ["v1/inter-service-triggering"],
  },

  // -------------------------------------------------------------------------
  // Flow control
  // -------------------------------------------------------------------------
  {
    name: "concurrency",
    query: "concurrency",
    expectAnyOf: ["v1/concurrency"],
  },
  {
    name: "rate limit",
    query: "rate limit",
    expectAnyOf: ["v1/rate-limits"],
  },
  {
    name: "rate limits (plural)",
    query: "rate limits",
    expectAnyOf: ["v1/rate-limits"],
  },
  {
    name: "priority",
    query: "priority",
    expectAnyOf: ["v1/priority"],
  },

  // -------------------------------------------------------------------------
  // Orchestration & composition
  // -------------------------------------------------------------------------
  {
    name: "orchestration",
    query: "orchestration",
    expectAnyOf: ["v1/durable-execution", "v1/patterns"],
    topN: 10,
  },
  {
    name: "DAG",
    query: "DAG",
    expectAnyOf: ["v1/durable-execution", "v1/patterns/directed-acyclic-graphs"],
    topN: 10,
  },
  {
    name: "conditional workflows",
    query: "conditional workflows",
    expectAnyOf: ["v1/durable-execution", "v1/conditions"],
    topN: 10,
  },
  {
    name: "on failure",
    query: "on failure",
    expectAnyOf: ["v1/durable-execution", "v1/on-failure", "v1/retry-policies"],
    topN: 10,
  },
  {
    name: "child spawning",
    query: "child spawning",
    expectAnyOf: ["v1/durable-execution", "v1/child-spawning"],
    topN: 10,
  },
  {
    name: "child tasks",
    query: "child tasks",
    expectAnyOf: ["v1/durable-execution", "v1/child-spawning", "v1/advanced-assignment/sticky-assignment"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // Durability
  // -------------------------------------------------------------------------
  {
    name: "durable execution",
    query: "durable execution",
    expectAnyOf: ["v1/durable-execution", "v1/patterns/durable-task-execution"],
  },
  {
    name: "durable events",
    query: "durable events",
    expectAnyOf: ["v1/durable-execution", "v1/events"],
  },
  {
    name: "durable sleep",
    query: "durable sleep",
    expectAnyOf: ["v1/durable-execution", "v1/sleep"],
  },
  {
    name: "durable best practices",
    query: "durable best practices",
    expectAnyOf: ["v1/durable-execution"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // Reliability & error handling
  // -------------------------------------------------------------------------
  {
    name: "retry",
    query: "retry",
    expectAnyOf: ["v1/retry-policies"],
  },
  {
    name: "timeout",
    query: "timeout",
    expectAnyOf: ["v1/timeouts"],
  },
  {
    name: "cancellation",
    query: "cancellation",
    expectAnyOf: ["v1/cancellation"],
  },
  {
    name: "bulk retries",
    query: "bulk retries",
    expectAnyOf: ["v1/bulk-retries-and-cancellations"],
  },

  // -------------------------------------------------------------------------
  // Worker management
  // -------------------------------------------------------------------------
  {
    name: "sticky assignment",
    query: "sticky assignment",
    expectAnyOf: ["v1/advanced-assignment/sticky-assignment"],
  },
  {
    name: "worker affinity",
    query: "worker affinity",
    expectAnyOf: ["v1/advanced-assignment/worker-affinity"],
  },
  {
    name: "manual slot release",
    query: "manual slot release",
    expectAnyOf: ["v1/advanced-assignment/manual-slot-release"],
  },
  {
    name: "autoscaling workers",
    query: "autoscaling workers",
    expectAnyOf: ["v1/autoscaling-workers", "v1/runtime/autoscaling-workers"],
  },
  {
    name: "worker health check",
    query: "worker health check",
    expectAnyOf: ["v1/worker-healthchecks"],
    topN: 10,
  },
  {
    name: "v1/troubleshooting",
    query: "v1/troubleshooting",
    expectAnyOf: ["v1/troubleshooting", "v1/troubleshooting/index"],
  },

  // -------------------------------------------------------------------------
  // Observability
  // -------------------------------------------------------------------------
  {
    name: "logging",
    query: "logging",
    expectAnyOf: ["v1/logging"],
  },
  {
    name: "opentelemetry",
    query: "opentelemetry",
    expectAnyOf: ["v1/opentelemetry"],
  },
  {
    name: "prometheus metrics",
    query: "prometheus metrics",
    expectAnyOf: ["self-hosting/prometheus-metrics", "v1/prometheus-metrics"],
  },
  {
    name: "streaming",
    query: "streaming",
    expectAnyOf: ["v1/streaming"],
  },
  {
    name: "additional metadata",
    query: "additional metadata",
    expectAnyOf: ["v1/additional-metadata", "v1/bulk-retries-and-cancellations"],
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
    expectAnyOf: ["v1/migrating/migration-guide-python"],
    skip: true,
  },
  {
    name: "migration typescript",
    query: "migration typescript",
    expectAnyOf: ["v1/migrating/migration-guide-typescript"],
    skip: true,
  },
  {
    name: "migration go",
    query: "migration go",
    expectAnyOf: ["v1/migrating/migration-guide-go"],
    skip: true,
  },
  {
    name: "engine migration",
    query: "engine migration",
    expectAnyOf: ["v1/migrating/migration-guide-engine"],
    skip: true,
  },
  {
    name: "SDK improvements",
    query: "SDK improvements",
    expectAnyOf: ["v1/migrating/v1-sdk-improvements"],
    skip: true,
  },

  // -------------------------------------------------------------------------
  // Self-hosting & infrastructure
  // -------------------------------------------------------------------------
  {
    name: "docker compose",
    query: "docker compose",
    expectAnyOf: ["self-hosting/docker-compose", "v1/docker", "v1/runtime/docker"],
  },
  {
    name: "running with docker",
    query: "running with docker",
    expectAnyOf: ["v1/docker", "v1/runtime/docker", "self-hosting/docker-compose"],
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
    expectAnyOf: ["reference/cli", "v1/developer-experience"],
  },
  {
    name: "TUI",
    query: "TUI",
    expectAnyOf: ["reference/cli", "reference/cli/tui"],
    topN: 10,
  },
  {
    name: "profiles",
    query: "profiles",
    expectAnyOf: ["reference/cli", "reference/cli/profiles"],
    topN: 10,
  },
  {
    name: "running hatchet locally",
    query: "running hatchet locally",
    expectAnyOf: ["reference/cli", "self-hosting/hatchet-lite", "v1/quickstart"],
    topN: 10,
  },

  // -------------------------------------------------------------------------
  // Code-specific searches
  // -------------------------------------------------------------------------
  {
    name: "SimpleInput — Pydantic model",
    query: "SimpleInput",
    expectAnyOf: ["v1/tasks"],
  },
  {
    name: "input_validator — Python arg",
    query: "input_validator",
    expectAnyOf: ["reference/python/pydantic", "v1/tasks"],
  },
  {
    name: "BaseModel — Pydantic",
    query: "BaseModel",
    expectAnyOf: ["reference/python/pydantic", "v1/tasks", "cookbooks/webhooks-stripe", "cookbooks/webhooks-github", "cookbooks/webhooks-slack"],
  },
  {
    name: "ctx.spawn — child spawn",
    query: "ctx.spawn",
    expectAnyOf: ["v1/durable-execution", "v1/child-spawning"],
    topN: 10,
  },
  {
    name: "NewStandaloneTask — Go API",
    query: "NewStandaloneTask",
    expectAnyOf: ["v1/tasks", "v1/migrating/migration-guide-go", "v1/external-events/run-on-event"],
  },
  {
    name: "DurableContext",
    query: "DurableContext",
    expectAnyOf: ["v1/durable-execution", "v1/patterns/durable-task-execution"],
    skip: true,
  },
  {
    name: "aio_run — Python async run",
    query: "aio_run",
    expectAnyOf: ["v1/tasks", "v1/running-your-task", "v1/bulk-run"],
  },

  // -------------------------------------------------------------------------
  // Special characters (regression tests)
  // -------------------------------------------------------------------------
  {
    name: "hatchet.task( — trailing paren",
    query: "hatchet.task(",
    expectAnyOf: ["v1/tasks"],
    topN: 10,
  },
  {
    name: "ctx.spawn( — trailing paren",
    query: "ctx.spawn(",
    expectAnyOf: ["v1/durable-execution", "v1/child-spawning"],
    topN: 10,
  },
  {
    name: ".run() — dot prefix and parens",
    query: ".run()",
    expectAnyOf: ["v1/tasks", "v1/running-your-task"],
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
    expectAnyOf: ["v1/durable-execution", "v1/sleep", "v1/scheduled-runs"],
  },
  {
    name: "debounce → concurrency",
    query: "debounce",
    expectAnyOf: ["v1/concurrency"],
  },
  {
    name: "dedup → concurrency",
    query: "dedup",
    expectAnyOf: ["v1/concurrency"],
  },
  {
    name: "throttle → rate limits",
    query: "throttle",
    expectAnyOf: ["v1/rate-limits", "v1/concurrency"],
  },
  {
    name: "fan out → child spawning",
    query: "fan out",
    expectAnyOf: ["v1/durable-execution", "v1/bulk-run", "v1/child-spawning"],
  },
  {
    name: "parallel tasks",
    query: "parallel tasks",
    expectAnyOf: ["v1/durable-execution", "v1/child-spawning"],
    topN: 10,
  },
  {
    name: "background job",
    query: "background job",
    expectAnyOf: ["v1/tasks", "v1/running-your-task", "v1/workers"],
  },
  {
    name: "recurring → cron",
    query: "recurring",
    expectAnyOf: ["v1/cron-runs"],
  },
  {
    name: "error handling → retry/failure",
    query: "error handling",
    expectAnyOf: ["v1/retry-policies", "v1/durable-execution", "v1/on-failure"],
    topN: 10,
  },
  {
    name: "fire and forget → run no wait",
    query: "fire and forget",
    expectAnyOf: ["v1/running-your-task"],
    topN: 10,
  },
  {
    name: "scale workers → autoscaling",
    query: "scale workers",
    expectAnyOf: ["v1/autoscaling-workers", "v1/runtime/autoscaling-workers"],
  },
  {
    name: "pipeline → DAG",
    query: "pipeline",
    expectAnyOf: [
      "v1/durable-execution",
      "v1/patterns/directed-acyclic-graphs",
      "cookbooks/rag-and-indexing",
      "cookbooks/document-processing",
    ],
    topN: 10,
  },
  {
    name: "long running task → durable",
    query: "long running task",
    expectAnyOf: ["v1/durable-execution", "v1/patterns/durable-task-execution", "v1/sleep"],
    topN: 10,
  },
  {
    name: "batch → bulk run",
    query: "batch tasks",
    expectAnyOf: ["v1/bulk-run", "cookbooks/batch-processing"],
    topN: 10,
  },
  {
    name: "if else → conditional",
    query: "if else workflow",
    expectAnyOf: ["v1/durable-execution", "v1/conditions"],
    topN: 10,
  },
  {
    name: "monitor → observability",
    query: "monitor",
    expectAnyOf: ["v1/opentelemetry", "v1/prometheus-metrics", "v1/logging", "self-hosting/prometheus-metrics"],
    topN: 10,
  },
  {
    name: "tracing → opentelemetry",
    query: "tracing",
    expectAnyOf: ["v1/opentelemetry"],
    topN: 10,
  },
  {
    name: "v1/observability",
    query: "v1/observability",
    expectAnyOf: ["v1/opentelemetry", "v1/prometheus-metrics", "v1/logging", "v1/streaming"],
    topN: 10,
    skip: true,
  },
  {
    name: "debug → troubleshooting",
    query: "debug",
    expectAnyOf: ["v1/troubleshooting", "v1/troubleshooting/index", "v1/logging"],
    topN: 10,
  },
  {
    name: "deploy → docker/k8s",
    query: "deploy",
    expectAnyOf: ["v1/docker", "v1/runtime/docker", "self-hosting/docker-compose", "self-hosting/kubernetes-quickstart"],
    topN: 10,
  },
  {
    name: "upgrade → migration",
    query: "upgrade",
    expectAnyOf: ["v1/migrating/migration-guide-python", "v1/migrating/migration-guide-typescript", "v1/migrating/migration-guide-go", "v1/migrating/migration-guide-engine", "self-hosting/upgrading-downgrading"],
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
    expectAnyOf: ["v1/worker-healthchecks"],
    topN: 10,
  },
  {
    name: "wait for event → durable events",
    query: "wait for event",
    expectAnyOf: ["v1/durable-execution", "v1/events", "v1/external-events/pushing-events", "v1/external-events/event-filters", "v1/sleep"],
    topN: 10,
  },
  {
    name: "api call → inter-service",
    query: "api call between services",
    expectAnyOf: ["v1/inter-service-triggering"],
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
    expectAnyOf: ["v1/retry-policies", "v1/durable-execution"],
    topN: 10,
  },
  {
    name: "how to run tasks in parallel",
    query: "how to run tasks in parallel",
    expectAnyOf: ["v1/durable-execution", "v1/child-spawning", "v1/running-your-task"],
    topN: 10,
  },
  {
    name: "how to cancel a running task",
    query: "how to cancel a running task",
    expectAnyOf: ["v1/cancellation"],
    topN: 10,
  },
  {
    name: "how to set up cron job",
    query: "how to set up cron job",
    expectAnyOf: ["v1/cron-runs"],
    topN: 10,
  },
  {
    name: "how to handle errors",
    query: "how to handle errors",
    expectAnyOf: ["v1/retry-policies", "v1/durable-execution", "v1/on-failure"],
    topN: 10,
  },
  {
    name: "how to limit concurrency",
    query: "how to limit concurrency",
    expectAnyOf: ["v1/concurrency", "v1/rate-limits"],
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
