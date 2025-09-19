export default {
  home: {
    title: "User Guide",
    type: "page",
    theme: {
      toc: false,
    },
  },
  _setup: {
    display: "hidden",
  },
  "self-hosting": {
    title: "Self Hosting",
    type: "page",
    theme: {
      toc: false,
    },
  },
  blog: {
    title: "Blog",
    type: "page",
  },
  contributing: {
    title: "Contributing",
    type: "page",
    display: "hidden",
    theme: {
      toc: false,
    },
  },
  sdks: {
    title: "SDK Reference",
    type: "menu",
    items: {
      python: {
        title: "Python",
        href: "/sdks/python/client",
        type: "page",
      },
      go: {
        title: "Go",
        href: "https://pkg.go.dev/github.com/hatchet-dev/hatchet/sdks/go",
        type: "page",
        newWindow: true,
      },
    },
  },
};
