import type { ReactNode } from "react";

export const metadata = { title: "Hatchet serverless on Next.js" };

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body style={{ fontFamily: "system-ui, sans-serif", maxWidth: 640, margin: "3rem auto" }}>
        {children}
      </body>
    </html>
  );
}
