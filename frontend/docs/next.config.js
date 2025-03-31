const withNextra = require('nextra')({
  theme: 'nextra-theme-docs',
  themeConfig: './theme.config.tsx',
})

/** @type {import('next').NextConfig} */
const nextConfig = {
  ...withNextra(),
  transpilePackages: ["react-tweet"],
  // swcMinify: false,
  async redirects() {
    return [
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

module.exports = nextConfig
