import type { AppProps } from "next/app";
// import { Inter } from "@next/font/google";
import posthog from "posthog-js";
import { PostHogProvider } from "posthog-js/react";

import "../styles/global.css";
import { useRouter } from "next/router";

// const inter = Inter({ subsets: ["latin"] });

// // Check that PostHog is client-side (used to handle Next.js SSR)
// if (typeof window !== "undefined" && process.env.NEXT_PUBLIC_POSTHOG_KEY) {
//   posthog.init(process.env.NEXT_PUBLIC_POSTHOG_KEY, {
//     // Enable debug mode in development
//     api_host: "https://docs.hatchet.run/ingest",
//     ui_host: "https://app.posthog.com",
//     loaded: (posthog) => {
//       if (process.env.NODE_ENV === "development") posthog.debug();
//     },
//   });
// }

function MyApp({ Component, pageProps }: AppProps) {
  return (
    <PostHogProvider client={posthog}>
      <main>
        <Component {...pageProps} />
      </main>
    </PostHogProvider>
  );
}

export default MyApp;
