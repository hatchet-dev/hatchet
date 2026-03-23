declare global {
  interface Window {
    __CONFIG__?: {
      BASE_PATH?: string;
    };
  }
}

export const config = {
  BASE_PATH: window.__CONFIG__?.BASE_PATH || '/',
};
