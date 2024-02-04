const withNextra = require('nextra')({
  theme: 'nextra-theme-docs',
  themeConfig: './theme.config.tsx',
})

module.exports = {
  ...withNextra(),
  async redirects() {
    return [
      {
        source: '/:path((?!home|contributing|favicon\\.ico).*)',
        destination: '/home/:path*',
        permanent: true,
      },
    ];
  },
}