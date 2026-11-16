import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // The adapter loads `ws` and `@vercel/functions` at runtime and the SDK carries a Node
  // client next to the core one; none of it belongs in the server bundle.
  serverExternalPackages: ["@hatchet-dev/serverless", "@hatchet-dev/typescript-sdk", "ws", "@vercel/functions"],
};

export default nextConfig;
