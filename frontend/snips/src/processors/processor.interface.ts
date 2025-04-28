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

type DirectoryProcessorProps = {
  dir: string;
};

export type ContentProcessor = (props: ContentProcessorProps) => Promise<ProcessedFile[]>;
export type DirectoryProcessor = (props: DirectoryProcessorProps) => Promise<void>;

export type Processor = {
  processFile: ContentProcessor;
  processDirectory: DirectoryProcessor;
};
