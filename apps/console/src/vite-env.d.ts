/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_NEKIRO_PUBLIC_AGENT_ORIGIN: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
