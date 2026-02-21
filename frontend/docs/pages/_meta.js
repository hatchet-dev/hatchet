export default {
  guide: {
    title: "Guide",
    type: "page",
    theme: {
      toc: false,
    },
  },
  patterns: {
    title: "Patterns",
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
  migrating: {
    title: "Migrating",
    type: "page",
    theme: {
      toc: false,
    },
  },
  reference: {
    title: "Reference",
    type: "menu",
    items: {
      cli: {
        title: "CLI Reference",
        href: "/cli",
        type: "page",
      },
      python: {
        title: "Python SDK",
        href: "/sdk/python/client",
        type: "page",
      },
      go: {
        title: "Go SDK",
        href: "https://pkg.go.dev/github.com/hatchet-dev/hatchet/sdks/go",
        type: "page",
        newWindow: true,
      }
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
  "agent-instructions": {
    title: "Agent Instructions",
    type: "page",
    display: "hidden",
    theme: {
      toc: false,
    },
  }
};
