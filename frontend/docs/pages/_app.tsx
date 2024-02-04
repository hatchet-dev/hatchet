import type { AppProps } from "next/app";
// import { Inter } from "@next/font/google";
import "../styles/global.css";

// const inter = Inter({ subsets: ["latin"] });

function MyApp({ Component, pageProps }: AppProps) {
  return (
    <>
      <main className="bg-[#020817]">
        <Component {...pageProps} />
      </main>
    </>
  );
}

export default MyApp;
