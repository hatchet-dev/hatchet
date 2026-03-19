import type { AppProps } from "next/app";
import "../styles/global.css";
import { LanguageProvider } from "../context/LanguageContext";
import { ConsentProvider } from "../context/ConsentContext";
import CookieConsent from "@/components/ui/cookie-banner";
import { PostHogProvider } from "@/providers/posthog";
import { CrossDomainLinkHandler } from "@/components/CrossDomainLinkHandler";
import { SidebarFolderNav } from "@/components/SidebarFolderNav";

function MyApp({ Component, pageProps }: AppProps) {
  return (
    <LanguageProvider>
      <ConsentProvider>
        <PostHogProvider>
          <CrossDomainLinkHandler>
            <main>
              <CookieConsent />
              <SidebarFolderNav />
              <Component {...pageProps} />
            </main>
          </CrossDomainLinkHandler>
        </PostHogProvider>
      </ConsentProvider>
    </LanguageProvider>
  );
}

export default MyApp;
