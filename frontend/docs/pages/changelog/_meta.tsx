import { LanguageLogo } from "@/lib/language-logos";

export default {
  index: "Overview",
  platform: {
    title: (
      <span className="flex items-center gap-2">
        <LanguageLogo language="go" size={20} /> Platform &amp; Go SDK
      </span>
    ),
  },
  python: {
    title: (
      <span className="flex items-center gap-2">
        <LanguageLogo language="py" /> Python SDK
      </span>
    ),
  },
  typescript: {
    title: (
      <span className="flex items-center gap-2">
        <LanguageLogo language="ts" /> TypeScript SDK
      </span>
    ),
  },
  ruby: {
    title: (
      <span className="flex items-center gap-2">
        <LanguageLogo language="rb" /> Ruby SDK
      </span>
    ),
  },
};
