type ContentProcessorProps = {
  path: string;
  name: string;
  content: string;
};

export type ContentProcessor = (props: ContentProcessorProps) => Promise<{
  filename?: string;
  content: string;
  outDir?: string;
}>;

export type Processor = {
  process: ContentProcessor;
  outDir?: string;
};
