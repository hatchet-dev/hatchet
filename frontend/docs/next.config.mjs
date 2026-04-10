// Using ESM for Nextra v4
import nextra from 'nextra'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

// Configure Nextra for MDX and docs
const withNextra = nextra({
  theme: 'nextra-theme-docs',
  themeConfig: './theme.config.tsx',
  defaultShowCopyCode: true,
  readingTime: true,
  staticImage: true,
  latex: false,
})

/** @type {import('next').NextConfig} */
const nextConfig = {
  webpack(config) {
    config.resolve.alias['@theguild/remark-mermaid/mermaid'] = path.resolve(
      __dirname,
      'components/Mermaid.tsx',
    )
    return config
  },
  transpilePackages: ["react-tweet"],
  swcMinify: false,
  images: {
    unoptimized: true,
  },
  async redirects() {
    return [
      // --- New site: section index redirects ---
      { source: '/', destination: '/v1', permanent: false, basePath: false },
      { source: '/get-started', destination: '/v1', permanent: false, basePath: false },
      { source: '/reference', destination: '/reference/python/client', permanent: false, basePath: false },
      { source: '/reference/', destination: '/reference/python/client', permanent: false, basePath: false },
      { source: '/reference/python', destination: '/reference/python/client', permanent: false, basePath: false },
      { source: '/reference/python/', destination: '/reference/python/client', permanent: false, basePath: false },
      { source: '/reference/typescript', destination: '/reference/typescript/client', permanent: false, basePath: false },
      { source: '/reference/typescript/', destination: '/reference/typescript/client', permanent: false, basePath: false },
      { source: '/v1/migrating', destination: '/v1/migrating/v1-sdk-improvements', permanent: false, basePath: false },
      { source: '/v1/migrating/', destination: '/v1/migrating/v1-sdk-improvements', permanent: false, basePath: false },
      { source: '/agent-instructions', destination: '/agent-instructions/setup-cli', permanent: false, basePath: false },
      { source: '/agent-instructions/', destination: '/agent-instructions/setup-cli', permanent: false, basePath: false },
      { source: '/reference/typescript/feature-clients', destination: '/reference/typescript/feature-clients/crons', permanent: false, basePath: false },
      { source: '/reference/typescript/feature-clients/', destination: '/reference/typescript/feature-clients/crons', permanent: false, basePath: false },
      { source: '/reference/python/feature-clients', destination: '/reference/python/feature-clients/cron', permanent: false, basePath: false },
      { source: '/reference/python/feature-clients/', destination: '/reference/python/feature-clients/cron', permanent: false, basePath: false },
      // --- Deleted v1 pages that were consolidated/renamed ---
      { source: '/v1/conditions', destination: '/v1/directed-acyclic-graphs#branching-with-parent-conditions', permanent: true, basePath: false },
      { source: '/v1/on-failure', destination: '/v1/error-handling', permanent: true, basePath: false },
      { source: '/v1/sleep', destination: '/v1/durable-sleep', permanent: true, basePath: false },
      { source: '/v1/external-events', destination: '/v1/events', permanent: true, basePath: false },
      { source: '/v1/external-events/:path*', destination: '/v1/events', permanent: true, basePath: false },
      { source: '/v1/patterns/directed-acyclic-graphs', destination: '/v1/directed-acyclic-graphs', permanent: true, basePath: false },
      { source: '/v1/patterns/durable-task-execution', destination: '/v1/durable-tasks', permanent: true, basePath: false },
      { source: '/v1/patterns/mixing-patterns', destination: '/v1/durable-execution', permanent: true, basePath: false },
      { source: '/v1/patterns', destination: '/v1/durable-execution', permanent: true, basePath: false },
      { source: '/v1/patterns/:path*', destination: '/v1/durable-execution', permanent: true, basePath: false },
      // --- Old main: /home/* → /v1/* (only paths that existed on main) ---
      { source: '/home/conditional-workflows', destination: '/v1/directed-acyclic-graphs#branching-with-parent-conditions', permanent: true, basePath: false },
      { source: '/home/on-failure-tasks', destination: '/v1/error-handling', permanent: true, basePath: false },
      { source: '/home/durable-execution', destination: '/v1/durable-execution', permanent: true, basePath: false },
      { source: '/home/:slug(dags|orchestration)', destination: '/v1/directed-acyclic-graphs', permanent: true, basePath: false },
      { source: '/home/durable-sleep', destination: '/v1/durable-sleep', permanent: true, basePath: false },
      { source: '/home/durable-events', destination: '/v1/events', permanent: true, basePath: false },
      { source: '/home/durable-best-practices', destination: '/v1/durable-execution', permanent: true, basePath: false },
      { source: '/home/architecture', destination: '/v1/architecture-and-guarantees', permanent: true, basePath: false },
      { source: '/home/your-first-task', destination: '/v1/tasks', permanent: true, basePath: false },
      { source: '/home/running-tasks', destination: '/v1/tasks', permanent: true, basePath: false },
      { source: '/home/setup', destination: '/v1/setup/advanced', permanent: true, basePath: false },
      { source: '/home/hatchet-cloud-quickstart', destination: '/v1/quickstart', permanent: true, basePath: false },
      { source: '/home/coding-agents', destination: '/v1/setup/using-coding-agents', permanent: true, basePath: false },
      { source: '/home/install-docs-mcp', destination: '/v1/setup/using-coding-agents', permanent: true, basePath: false },
      { source: '/home/guarantees-and-tradeoffs', destination: '/v1/architecture-and-guarantees', permanent: true, basePath: false },
      { source: '/home/v1-sdk-improvements', destination: '/v1/migrating/v1-sdk-improvements', permanent: true, basePath: false },
      { source: '/home/migration-guide-engine', destination: '/v1/migrating/migration-guide-engine', permanent: true, basePath: false },
      { source: '/home/migration-guide-python', destination: '/v1/migrating/migration-guide-python', permanent: true, basePath: false },
      { source: '/home/migration-guide-typescript', destination: '/v1/migrating/migration-guide-typescript', permanent: true, basePath: false },
      { source: '/home/migration-guide-go', destination: '/v1/migrating/migration-guide-go', permanent: true, basePath: false },
      { source: '/home/:slug(asyncio|pydantic|lifespans|dependency-injection|dataclasses)', destination: '/reference/python/:slug', permanent: true, basePath: false },
      // Old main had redirects from /home/basics/* and /home/features/* → ensure those source URLs still resolve
      { source: '/home/basics/overview', destination: '/v1/setup/advanced', permanent: true, basePath: false },
      { source: '/home/basics/(steps|workflows)', destination: '/v1/tasks', permanent: true, basePath: false },
      { source: '/home/basics/environments', destination: '/v1/setup/advanced/environments', permanent: true, basePath: false },
      { source: '/home/features/concurrency/:path*', destination: '/v1/concurrency', permanent: true, basePath: false },
      { source: '/home/features/durable-execution', destination: '/v1/durable-execution', permanent: true, basePath: false },
      { source: '/home/features/retries/:path*', destination: '/v1/retry-policies', permanent: true, basePath: false },
      { source: '/home/features/errors-and-logging', destination: '/v1/logging', permanent: true, basePath: false },
      { source: '/home/features/on-failure-step', destination: '/v1/error-handling', permanent: true, basePath: false },
      { source: '/home/features/triggering-runs/event-trigger', destination: '/v1/events', permanent: true, basePath: false },
      { source: '/home/features/triggering-runs/cron-trigger', destination: '/v1/cron-runs', permanent: true, basePath: false },
      { source: '/home/features/triggering-runs/schedule-trigger', destination: '/v1/scheduled-runs', permanent: true, basePath: false },
      { source: '/home/features/rate-limits', destination: '/v1/rate-limits', permanent: true, basePath: false },
      { source: '/home/features/worker-assignment/overview', destination: '/v1/advanced-assignment/sticky-assignment', permanent: true, basePath: false },
      { source: '/home/features/worker-assignment/(overview|sticky-assignment)', destination: '/v1/advanced-assignment/sticky-assignment', permanent: true, basePath: false },
      { source: '/home/features/worker-assignment/worker-affinity', destination: '/v1/advanced-assignment/worker-affinity', permanent: true, basePath: false },
      { source: '/home/features/additional-metadata', destination: '/v1/additional-metadata', permanent: true, basePath: false },
      { source: '/home/features/advanced/manual-slot-release', destination: '/v1/advanced-assignment/manual-slot-release', permanent: true, basePath: false },
      { source: '/home/features/opentelemetry', destination: '/v1/opentelemetry', permanent: true, basePath: false },
      { source: '/home/features/cancellation', destination: '/v1/cancellation', permanent: true, basePath: false },
      { source: '/home/features/child-workflows', destination: '/v1/child-spawning', permanent: true, basePath: false },
      { source: '/home/:path*', destination: '/v1/:path*', permanent: false, basePath: false },
      // Old main: /compute → /home/compute
      { source: '/compute', destination: '/v1/compute', permanent: true, basePath: false },
      { source: '/compute/:path*', destination: '/v1/compute', permanent: true, basePath: false },
      // --- Old main: sdks, cli, guides ---
      { source: '/sdks/python-sdk/:path*', destination: '/reference/python/client', permanent: false, basePath: false },
      { source: '/sdks/python', destination: '/reference/python/client', permanent: false, basePath: false },
      { source: '/sdks/:path*', destination: '/reference/:path*', permanent: false, basePath: false },
      { source: '/sdk/:path*', destination: '/reference/:path*', permanent: false, basePath: false },
      { source: '/cli/:path*', destination: '/reference/cli/:path*', permanent: false, basePath: false },
      { source: '/guides/:path*', destination: '/cookbooks/:path*', permanent: true, basePath: false },
      // --- Misc ---
      { source: '/ingest/:path*', destination: 'https://app.posthog.com/:path*', permanent: false, basePath: false },
      // Blog redirects to hatchet.run
      {
        source: "/blog/automated-documentation",
        destination: "https://hatchet.run/blog/automated-documentation",
        permanent: true,
        basePath: false,
      },
      {
        source: "/blog/background-tasks-fastapi-hatchet",
        destination: "https://hatchet.run/blog/fastapi-background-jobs-to-hatchet",
        permanent: true,
        basePath: false,
      },
      {
        source: "/blog/go-agents",
        destination: "https://hatchet.run/blog/go-agents",
        permanent: true,
        basePath: false,
      },
      {
        source: "/blog/warning-event-loop-blocked",
        destination: "https://hatchet.run/blog/warning-event-loop-blocked",
        permanent: true,
        basePath: false,
      },
      {
        source: "/blog/fastest-postgres-inserts",
        destination: "https://hatchet.run/blog/fastest-postgres-inserts",
        permanent: true,
        basePath: false,
      },
      {
        source: "/blog/task-queue-modern-python",
        destination: "https://hatchet.run/blog/task-queue-modern-python",
        permanent: true,
        basePath: false,
      },
      {
        source: "/blog/postgres-events-table",
        destination: "https://hatchet.run/blog/postgres-events-table",
        permanent: true,
        basePath: false,
      },
      {
        source: "/blog/migrating-off-prisma",
        destination: "https://hatchet.run/blog",
        permanent: true,
        basePath: false,
      },
      {
        source: "/blog/problems-with-celery",
        destination: "https://hatchet.run/blog/problems-with-celery",
        permanent: true,
        basePath: false,
      },
      {
        source: "/blog/multi-tenant-queues",
        destination: "https://hatchet.run/blog/multi-tenant-queues",
        permanent: true,
        basePath: false,
      },
      {
        source: "/blog/mergent-migration-guide",
        destination: "https://hatchet.run/blog",
        permanent: true,
        basePath: false,
      },
      {
        source: "/blog",
        destination: "https://hatchet.run/blog",
        permanent: true,
        basePath: false,
      },
    ];
  },
}

export default withNextra(nextConfig)
