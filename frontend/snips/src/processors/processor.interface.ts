type ContentProcessorProps = {
  path: string;
  name: string;
  content: string;
};

export type ContentProcessor = (props: ContentProcessorProps) => Promise<{
  filename?: string;
  content: string;
}>;

export type Processor = ContentProcessor;
