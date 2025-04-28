type ContentProcessorProps = {
  path: string;
  name: string;
  content: string;
};

type ProcessedFile = {
  filename?: string;
  content: string;
  outDir?: string;
};

export type ContentProcessor = (props: ContentProcessorProps) => Promise<ProcessedFile[]>;

export type Processor = {
  process: ContentProcessor;
};
