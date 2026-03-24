export default {
  v1: {
    title: "Guide",
    type: "page",
    theme: {
      toc: true,
    },
  },
  cookbooks: {
    title: "Cookbooks",
    type: "page",
    theme: {
      toc: true,
    },
  },
  "self-hosting": {
    title: "Self-Hosting",
    type: "page",
    theme: {
      toc: true,
    },
  },
  reference: {
    title: "Reference",
    type: "menu",
    items: {
      cli: {
        title: "CLI Reference",
        href: "/reference/cli",
        type: "page",
      },
      python: {
        title: "Python SDK",
        href: "/reference/python/client",
        type: "page",
      },
      typescript: {
        title: "Typescript SDK",
        href: "/reference/typescript/client",
        type: "page",
      },
      go: {
        title: "Go SDK",
        href: "https://pkg.go.dev/github.com/hatchet-dev/hatchet/sdks/go",
        type: "page",
        newWindow: true,
      },
    },
  },
  contributing: {
    title: "Contributing",
    type: "page",
    display: "hidden",
    theme: {
      toc: true,
    },
  },
  "agent-instructions": {
    title: "Agent Instructions",
    type: "page",
    display: "hidden",
    theme: {
      toc: false,
    },
  },
};
