export type Highlight = {
  lines: number[];
  strings: string[];
};

export type Block = {
  start: number;
  stop: number;
};

// Types for snippets
export type Snippet = {
  content: string;
  language: string;
  source: string;
  blocks?: {
    [key: string]: Block;
  };
  highlights?: {
    [key: string]: Highlight;
  };
};

export const LANGUAGE_MAP = {
  ts: 'typescript ',
  py: 'python',
  go: 'go',
  unknown: 'unknown',
};

export default {};
