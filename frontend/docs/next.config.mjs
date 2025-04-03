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
  transpilePackages: ["react-tweet"],
  swcMinify: false,
  images: {
    unoptimized: true,
  },
  async redirects() {
    return [
      {
        source: '/compute',
        destination: '/home/compute',
        permanent: true,
      },
      {
        source: '/compute/:path',
        destination: '/home/compute',
        permanent: true,
      },
      {
        source: '/:path((?!home|v1|v0|compute|sdk|contributing|self-hosting|launches|blog|favicon\\.ico|hatchet_logo\\.png|_next/.*|monitoring\-demo\.mp4).*)',
        destination: '/home/:path*',
        permanent: false,
      },
      {
        source: "/ingest/:path*",
        destination: "https://app.posthog.com/:path*",
        permanent: false,
      },
      {
        source: "/home/basics/overview",
        destination: "/home/setup",
        permanent: false,
      },
      {
        source: "/home/basics/(steps|workflows)",
        destination: "/home/your-first-task",
        permanent: false,
      },
      {
        source: "/home/basics/environments",
        destination: "/home/environments",
        permanent: false,
      },
      {
        source: "/home/features/concurrency/:path*",
        destination: "/home/concurrency",
        permanent: false,
      },
      {
        source: "/home/features/durable-execution",
        destination: "/home/durable-execution",
        permanent: false,
      },
      {
        source: "/home/features/retries/:path*",
        destination: "/home/retry-policies",
        permanent: false,
      },
      {
        source: "/home/features/errors-and-logging",
        destination: "/home/logging",
        permanent: false,
      },
      {
        source: "/home/features/on-failure-step",
        destination: "/home/on-failure-tasks",
        permanent: false,
      },
      {
        source: "/home/features/triggering-runs/event-trigger",
        destination: "/home/run-on-event",
        permanent: false,
      },
      {
        source: "/home/features/triggering-runs/cron-trigger",
        destination: "/home/cron-runs",
        permanent: false,
      },
      {
        source: "/home/features/triggering-runs/schedule-trigger",
        destination: "/home/scheduled-runs",
        permanent: false,
      },
      {
        source: "/home/features/rate-limits",
        destination: "/home/rate-limits",
        permanent: false,
      },
      {
        source: "/home/features/worker-assignment/overview",
        destination: "/home/sticky-assignment",
        permanent: false,
      },
      {
        source: "/home/features/worker-assignment/(overview|sticky-assignment)",
        destination: "/home/sticky-assignment",
        permanent: false,
      },
      {
        source: "/home/features/worker-assignment/worker-affinity",
        destination: "/home/worker-affinity",
        permanent: false,
      },
      {
        source: "/home/features/additional-metadata",
        destination: "/home/additional-metadata",
        permanent: false,
      },
      {
        source: "/home/features/advanced/manual-slot-release",
        destination: "/home/manual-slot-release",
        permanent: false,
      },
      {
        source: "/home/features/opentelemetry",
        destination: "/home/opentelemetry",
        permanent: false,
      },
      {
        source: "/home/features/cancellation",
        destination: "/home/cancellation",
        permanent: false,
      },
      {
        source: "/home/features/child-workflows",
        destination: "/home/child-spawning",
        permanent: false,
      },
    ];
  },
}

export default withNextra(nextConfig)
