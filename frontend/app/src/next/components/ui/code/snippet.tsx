import { CodeBlock } from './code-block';
import { snippets } from '@/next/lib/docs/snips';

interface GithubSnippetProps {
  src: string;
  highlightLines?: number[];
  showLineNumbers?: boolean;
  title?: string;
}

// This is a server component that will be rendered at build time
export const Snippet = ({ src, ...props }: GithubSnippetProps) => {
  if (!src) {
    return null;
  }

  const [, filePath] = src.split(':');

  // Get the snippet content from the snippets object
  const snippet = snippets[filePath];
  if (!snippet) {
    throw new Error(`Snippet content not found: ${filePath}`);
  }

  return (
    <>
      <CodeBlock
        value={snippet.content}
        language={snippet.language}
        //   highlightLines={snippet.highlights}
        {...props}
        title={props.title || snippet.source}
        link={`https://github.com/hatchet-dev/hatchet/blob/main/${snippet.source}`}
      />
    </>
  );
};
