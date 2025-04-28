// Types for snippets
export type Snippet = {
  content: string;
  language: string;
  source: string;
  highlights?: {
    [key: string]: {
      lines: number[];
      strings: string[];
    };
  };
};

export const LANGUAGE_MAP = {
  ts: 'typescript ',
  py: 'python',
  go: 'go',
  unknown: 'unknown',
};
