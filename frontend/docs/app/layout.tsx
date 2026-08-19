import "@/styles/global.css";
import { Inter } from "next/font/google";
import { RootProvider } from "fumadocs-ui/provider/next";
import { Suspense, type ReactNode } from "react";
import { LanguageProvider } from "@/context/LanguageContext";
import { ConsentProvider } from "@/context/ConsentContext";
import { PostHogProvider } from "@/providers/posthog";
import CookieConsent from "@/components/ui/cookie-banner";
import { ThemeQueryParam } from "@/components/ThemeQueryParam";
import HatchetSearchDialog from "@/components/HatchetSearchDialog";

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
});

export const metadata = {
  title: {
    default: "Hatchet Documentation",
    template: "%s - Hatchet Documentation",
  },
};

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html
      lang="en"
      className={`dark ${inter.variable}`}
      suppressHydrationWarning
    >
      <head>
        <link rel="icon" type="image/png" href="/favicon.ico" />
        <link rel="prefetch" href="/llms-search-index.json" />
      </head>
      <body>
        <LanguageProvider>
          <ConsentProvider>
            <PostHogProvider>
              <RootProvider
                theme={{ defaultTheme: "dark" }}
                search={{ SearchDialog: HatchetSearchDialog }}
              >
                <CookieConsent />
                <Suspense fallback={null}>
                  <ThemeQueryParam />
                </Suspense>
                {children}
              </RootProvider>
            </PostHogProvider>
          </ConsentProvider>
        </LanguageProvider>
      </body>
    </html>
  );
}
