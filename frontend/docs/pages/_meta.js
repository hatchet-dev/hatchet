export default {
  "home": {
    "title": "User Guide",
    "type": "page"
  },
  "_setup": {
    "display": "hidden"
  },
  "compute": {
    "title": "Managed Compute",
    "type": "page",
    "href": "/home/compute",
    "index": "Overview",
    "getting-started": "Getting Started",
    "cpu": "CPU Machine Types",
    "gpu": "GPU Machine Types",
    "-- SDKs": {
      "type": "separator",
      "title": "SDK Deployment Guides"
    },
    "python": {
      "title": "Python ↗",
      "href": "/sdks/python-sdk/docker"
    },
    "typescript": {
      "title": "TypeScript ↗",
      "href": "/sdks/typescript-sdk/docker"
    }
  },
  "self-hosting": {
    "title": "Self Hosting",
    "type": "page"
  },
  "blog": {
    "title": "Blog",
    "type": "page"
  },
  "contributing": {
    "title": "Contributing",
    "type": "page",
    "display": "hidden"
  },
  "sdks": {
    "title": "SDK Reference",
    "type": "menu",
    "items": {
      "python": {
        "title": "Python SDK",
        "type": "page",
        "items": {
          "index": "Overview",
          "api": "API Reference"
        }
      },
      "typescript": {
        "title": "TypeScript SDK",
        "type": "page",
        "items": {
          "index": "Overview",
          "api": "API Reference"
        }
      }
    }
  },
  "v0": {
    "title": "V0 (Old docs)",
    "type": "page",
    "href": "https://v0-docs.hatchet.run"
  }
}
