const withNextra = require('nextra')({
  theme: 'nextra-theme-docs',
  themeConfig: './theme.config.tsx',
})

/** @type {import('next').NextConfig} */
const nextConfig = {
  ...withNextra(),
  transpilePackages: ["react-tweet"],
  async redirects() {
    return [
      {
        source: '/:path((?!home|sdk|contributing|self-hosting|launches|favicon\\.ico|hatchet_logo\\.png).*)',
        destination: '/home/:path*',
        permanent: true,
      },
      {
        source: "/ingest/:path*",
        destination: "https://app.posthog.com/:path*",
        permanent: true,
      },
    ];
  },
}

module.exports = nextConfig
