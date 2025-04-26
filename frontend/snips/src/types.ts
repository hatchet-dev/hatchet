// Types for snippets
export type Snippet = {
  content: string;
};

export type EXTRA = {
  language: string;
  source: string;
  highlights?: {
    [key: string]: {
      lines: number[];
      strings: string[];
    };
  };
};
