import type { AppProps } from "next/app";
import posthog from "posthog-js";
import { PostHogProvider } from "posthog-js/react";
import "../styles/global.css";
import { LanguageProvider } from "context/LanguageContext";

function MyApp({ Component, pageProps }: AppProps) {
  return (
    <LanguageProvider>
      <PostHogProvider client={posthog}>
        <main>
          <Component {...pageProps} />
        </main>
      </PostHogProvider>
    </LanguageProvider>
  );
}

export default MyApp;
