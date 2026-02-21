// Using ESM for Nextra v4
import nextra from 'nextra'

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
  basePath: '/v1',
  transpilePackages: ["react-tweet"],
  swcMinify: false,
  images: {
    unoptimized: true,
  },
  async redirects() {
    return [
      {
        source: '/',
        destination: '/v1/guide',
        permanent: false,
        basePath: false,
      },
      {
        source: '/',
        destination: '/guide',
        permanent: false,
      },
      {
        source: '/guide',
        destination: '/guide/what-is-hatchet',
        permanent: false,
      },
      {
        source: '/guide/setup',
        destination: '/guide/hatchet-cloud-quickstart/setup',
        permanent: true,
      },
      {
        source: '/guide/environments',
        destination: '/guide/hatchet-cloud-quickstart/environments',
        permanent: true,
      },
      {
        source: '/compute',
        destination: '/v1/guide/compute',
        permanent: true,
        basePath: false,
      },
      {
        source: '/compute/:path',
        destination: '/v1/guide/compute',
        permanent: true,
        basePath: false,
      },
      {
        source: '/:path((?!api|agent-instructions|home|cli|v1|v0|compute|sdk|contributing|self-hosting|launches|blog|llms|favicon\\.ico|.*\\.png|.*\\.gif|.*\\.svg|_next/.*|monitoring\-demo\.mp4).*)',
        destination: '/home/:path*',
        permanent: false,
      },
      {
        source: "/ingest/:path*",
        destination: "https://app.posthog.com/:path*",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/install-docs-mcp",
        destination: "/home/coding-agents",
        permanent: true,
      },
      {
        source: "/home/basics/overview",
        destination: "/v1/guide/hatchet-cloud-quickstart/setup",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/basics/(steps|workflows)",
        destination: "/v1/guide/your-first-task",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/basics/environments",
        destination: "/v1/guide/hatchet-cloud-quickstart/environments",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/concurrency/:path*",
        destination: "/v1/features/concurrency",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/durable-execution",
        destination: "/v1/features/durable-execution",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/retries/:path*",
        destination: "/v1/features/retry-policies",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/errors-and-logging",
        destination: "/v1/features/logging",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/on-failure-step",
        destination: "/v1/features/on-failure-tasks",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/triggering-runs/event-trigger",
        destination: "/v1/features/run-on-event",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/triggering-runs/cron-trigger",
        destination: "/v1/features/cron-runs",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/triggering-runs/schedule-trigger",
        destination: "/v1/features/scheduled-runs",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/rate-limits",
        destination: "/v1/features/rate-limits",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/worker-assignment/overview",
        destination: "/v1/features/sticky-assignment",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/worker-assignment/(overview|sticky-assignment)",
        destination: "/v1/features/sticky-assignment",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/worker-assignment/worker-affinity",
        destination: "/v1/features/worker-affinity",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/additional-metadata",
        destination: "/v1/features/additional-metadata",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/advanced/manual-slot-release",
        destination: "/v1/features/manual-slot-release",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/opentelemetry",
        destination: "/v1/features/opentelemetry",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/cancellation",
        destination: "/v1/features/cancellation",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/features/child-workflows",
        destination: "/v1/features/child-spawning",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/:slug(concurrency|rate-limits|priority|orchestration|dags|conditional-workflows|on-failure-tasks|child-spawning|additional-metadata|durable-execution|durable-events|durable-sleep|durable-best-practices|timeouts|retry-policies|bulk-retries-and-cancellations|sticky-assignment|worker-affinity|manual-slot-release|logging|opentelemetry|prometheus-metrics|cancellation|streaming)",
        destination: "/v1/features/:slug",
        permanent: true,
        basePath: false,
      },
      {
        source: "/:slug(concurrency|rate-limits|priority|orchestration|dags|conditional-workflows|on-failure-tasks|child-spawning|additional-metadata|durable-execution|durable-events|durable-sleep|durable-best-practices|timeouts|retry-policies|bulk-retries-and-cancellations|sticky-assignment|worker-affinity|manual-slot-release|logging|opentelemetry|prometheus-metrics|cancellation|streaming)",
        destination: "/v1/features/:slug",
        permanent: true,
        basePath: false,
      },
      {
        source: "/home/:slug(asyncio|pydantic|lifespans|dependency-injection|dataclasses)",
        destination: "/v1/sdk/python/:slug",
        permanent: true,
        basePath: false,
      },
      {
        source: "/:slug(asyncio|pydantic|lifespans|dependency-injection|dataclasses)",
        destination: "/v1/sdk/python/:slug",
        permanent: true,
        basePath: false,
      },
      {
        source: "/sdks/python-sdk/:path*",
        destination: "/v1/sdk/python/client",
        permanent: false,
        basePath: false,
      },
      {
        source: "/sdks/python",
        destination: "/v1/sdk/python/client",
        permanent: false,
        basePath: false,
      },
      {
        source: "/sdks/:path*",
        destination: "/v1/sdk/:path*",
        permanent: false,
        basePath: false,
      },
      {
        source: "/sdk/:path*",
        destination: "/v1/sdk/:path*",
        permanent: false,
        basePath: false,
      },
      {
        source: "/cli/:path*",
        destination: "/v1/cli/:path*",
        permanent: false,
        basePath: false,
      },
      {
        source: "/self-hosting/:path*",
        destination: "/v1/self-hosting/:path*",
        permanent: false,
        basePath: false,
      },
      {
        source: "/features/:path*",
        destination: "/v1/features/:path*",
        permanent: false,
        basePath: false,
      },
      {
        source: "/guide/:path*",
        destination: "/v1/guide/:path*",
        permanent: false,
        basePath: false,
      },
      {
        source: "/guide/patterns",
        destination: "/v1/patterns/fanout",
        permanent: false,
        basePath: false,
      },
      {
        source: "/guide/patterns/:path*",
        destination: "/v1/patterns/:path*",
        permanent: false,
        basePath: false,
      },
      {
        source: "/guide/use-cases",
        destination: "/v1/patterns/rag-and-indexing",
        permanent: false,
        basePath: false,
      },
      {
        source: "/guide/use-cases/:path*",
        destination: "/v1/patterns/:path*",
        permanent: false,
        basePath: false,
      },
      {
        source: "/guide/v1-sdk-improvements",
        destination: "/v1/migrating/v0-to-v1/v1-sdk-improvements",
        permanent: false,
        basePath: false,
      },
      {
        source: "/guide/migration-guide-engine",
        destination: "/v1/migrating/v0-to-v1/migration-guide-engine",
        permanent: false,
        basePath: false,
      },
      {
        source: "/guide/migration-guide-python",
        destination: "/v1/migrating/v0-to-v1/migration-guide-python",
        permanent: false,
        basePath: false,
      },
      {
        source: "/guide/migration-guide-typescript",
        destination: "/v1/migrating/v0-to-v1/migration-guide-typescript",
        permanent: false,
        basePath: false,
      },
      {
        source: "/guide/migration-guide-go",
        destination: "/v1/migrating/v0-to-v1/migration-guide-go",
        permanent: false,
        basePath: false,
      },
      {
        source: "/migrations/:path*",
        destination: "/v1/migrating/v0-to-v1/:path*",
        permanent: false,
        basePath: false,
      },
      {
        source: "/home/:path*",
        destination: "/v1/guide/:path*",
        permanent: false,
        basePath: false,
      },
      {
        source: '/:path((?!api|ingest|v1|v0|compute|home|guide|features|patterns|migrating|cli|sdk|sdks|contributing|self-hosting|launches|blog|llms|favicon\\.ico|.*\\.png|.*\\.gif|.*\\.svg|_next/.*|monitoring\\-demo\\.mp4).*)',
        destination: '/v1/guide/:path*',
        permanent: false,
        basePath: false,
      },
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
      }
    ];
  },
}

export default withNextra(nextConfig)
