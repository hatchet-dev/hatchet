import Link from "next/link";
import { useEffect, useState } from "react";

const Logo = ({ color }: { color: string }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 198 49"
    width="140"
    height="40"
    fill="none"
  >
    <path
      fill={color}
      d="m25.22 21.618 6.486-14.412 12.402 12.402c1.713 1.713 2.569 2.569 2.89 3.556a4.324 4.324 0 0 1 0 2.672c-.321.987-1.177 1.843-2.89 3.556L31.706 41.794H21.618l16.936-18.239c.568-.612.852-.918.866-1.179a.72.72 0 0 0-.258-.59c-.2-.168-.618-.168-1.453-.168H25.221ZM23.78 27.382l-6.486 14.412L4.892 29.392c-1.713-1.713-2.569-2.569-2.89-3.556a4.324 4.324 0 0 1 0-2.672c.321-.987 1.177-1.843 2.89-3.556L17.294 7.207h10.088L10.447 25.445c-.568.612-.852.918-.866 1.179a.72.72 0 0 0 .258.59c.2.168.618.168 1.453.168H23.78ZM186.5 36.963v-20.22h-5.8v-4.64h16.573v4.64h-5.801v20.22H186.5ZM162.725 36.93V12.07h14.253v4.64h-9.281v5.171h7.856v4.641h-7.856v5.768h9.347v4.64h-14.319ZM140.813 36.93V12.07h4.972v10.077h7.193V12.07h4.972v24.86h-4.972V26.787h-7.193V36.93h-4.972ZM127.675 37.262c-5.039 0-8.585-3.182-8.585-8.52v-8.617c0-5.27 3.58-8.387 8.585-8.387 5.005 0 8.585 3.116 8.585 8.387v1.657h-4.972v-1.525c0-2.353-1.459-3.878-3.613-3.878-2.155 0-3.613 1.525-3.613 3.878v8.486c0 2.353 1.458 3.878 3.613 3.878 2.154 0 3.613-1.525 3.613-3.878V26.82h4.972v1.923c0 5.337-3.547 8.519-8.585 8.519ZM105.132 36.963v-20.22h-5.8v-4.64h16.573v4.64h-5.801v20.22h-4.972ZM79.94 36.93l6.165-24.86h6.397l6.166 24.86H92.8l-.663-3.348h-6.033l-.663 3.348H79.94Zm6.96-7.325h4.475l-2.22-11.27-2.254 11.27ZM59.088 36.93V12.07h4.973v10.077h7.192V12.07h4.972v24.86h-4.972V26.787h-7.192V36.93h-4.973Z"
    />
  </svg>
);

function useDarkMode() {
  const [dark, setDark] = useState(true);

  useEffect(() => {
    const stored = localStorage.getItem("theme");
    if (stored === "light") {
      setDark(false);
    } else if (stored === "dark") {
      setDark(true);
    } else if (stored === "system" || !stored) {
      setDark(window.matchMedia("(prefers-color-scheme: dark)").matches);
    }
  }, []);

  return dark;
}

const light = {
  bg: "#ffffff",
  fg: "#0f172a",
  muted: "#64748b",
  cardBg: "rgba(0,0,0,0.02)",
  border: "rgba(0,0,0,0.1)",
  hoverBg: "rgba(0,0,0,0.05)",
  logo: "hsl(228 61% 10%)",
};

const darkTheme = {
  bg: "#02081d",
  fg: "#e2e8f0",
  muted: "#94a3b8",
  cardBg: "rgba(255,255,255,0.04)",
  border: "rgba(255,255,255,0.1)",
  hoverBg: "rgba(255,255,255,0.08)",
  logo: "hsl(212 100% 86%)",
};

export default function Custom404() {
  const dark = useDarkMode();
  const t = dark ? darkTheme : light;

  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        padding: "3rem 1.5rem",
        background: t.bg,
        color: t.fg,
        fontFamily:
          '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, sans-serif',
      }}
    >
      <Link href="/" style={{ marginBottom: "2.5rem" }}>
        <Logo color={t.logo} />
      </Link>

      <h1
        style={{
          fontSize: "5rem",
          lineHeight: 1,
          fontWeight: 700,
          color: t.muted,
          margin: "0 0 0.75rem",
          letterSpacing: "-0.04em",
        }}
      >
        404
      </h1>

      <p
        style={{
          fontSize: "1.125rem",
          color: t.muted,
          margin: "0 0 2.5rem",
          maxWidth: "28rem",
          textAlign: "center",
        }}
      >
        This page doesn&apos;t exist. It may have been moved or removed.
      </p>

      <Link
        href="/v1"
        style={{
          display: "inline-flex",
          alignItems: "center",
          gap: "0.5rem",
          padding: "0.75rem 1.5rem",
          borderRadius: "0.5rem",
          border: `1px solid ${t.border}`,
          background: t.cardBg,
          color: t.fg,
          fontSize: "0.9375rem",
          fontWeight: 600,
          textDecoration: "none",
          transition: "background-color 0.15s ease, border-color 0.15s ease",
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.background = t.hoverBg;
          e.currentTarget.style.borderColor = t.muted;
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.background = t.cardBg;
          e.currentTarget.style.borderColor = t.border;
        }}
      >
        &larr; Back to Home
      </Link>
    </div>
  );
}
