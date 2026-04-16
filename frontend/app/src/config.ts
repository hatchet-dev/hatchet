declare global {
  interface Window {
    __CONFIG__?: {
      BASE_PATH?: string;
    };
  }
}

export const config = {
  get BASE_PATH() {
    return window.__CONFIG__?.BASE_PATH || '/';
  },
};
