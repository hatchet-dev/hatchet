import nextra from 'nextra';

const withNextra = nextra({
  theme: 'nextra-theme-docs',
  themeConfig: './theme.config.tsx',
});

/** @type {import('next').NextConfig} */
const nextConfig = {
  ...withNextra(),
  transpilePackages: ["react-tweet"],
  async redirects() {
    return [
      {
        source: '/:path((?!home|compute|sdk|contributing|self-hosting|launches|blog|favicon\\.ico|hatchet_logo\\.png|_next/.*|monitoring\-demo\.mp4).*)',
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
};

export default nextConfig;
