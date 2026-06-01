import type { AppProps } from "next/app";
import "../styles/global.css";
import { LanguageProvider } from "../context/LanguageContext";
import { ConsentProvider } from "../context/ConsentContext";
import CookieConsent from "@/components/ui/cookie-banner";
import { PostHogProvider } from "@/providers/posthog";
import { SidebarFolderNav } from "@/components/SidebarFolderNav";

function MyApp({ Component, pageProps }: AppProps) {
  return (
    <LanguageProvider>
      <ConsentProvider>
        <PostHogProvider>
          <main>
            <CookieConsent />
            <SidebarFolderNav />
            <Component {...pageProps} />
          </main>
        </PostHogProvider>
      </ConsentProvider>
    </LanguageProvider>
  );
}

export default MyApp;
