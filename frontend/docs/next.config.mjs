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
    ];
  },
}

export default withNextra(nextConfig)
