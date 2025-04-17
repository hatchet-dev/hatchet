/// <reference types="vite/client" />
interface ImportMetaEnv {
  readonly VITE_SENTRY_DSN: string | undefined;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
