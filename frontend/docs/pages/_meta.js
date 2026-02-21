export default {
  guide: {
    title: "Guide",
    type: "page",
    theme: {
      toc: false,
    },
  },
  features: {
    title: "Features",
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
  contributing: {
    title: "Contributing",
    type: "page",
    display: "hidden",
    theme: {
      toc: false,
    },
  },
  cli: {
    title: "CLI Reference",
    type: "page",
    theme: {
      toc: false,
    },
  },
  sdk: {
    title: "SDK Reference",
    type: "menu",
    items: {
      python: {
        title: "Python",
        href: "/sdk/python/client",
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
