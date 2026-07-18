import { CodeHighlighter } from './code-highlighter';
import { cn } from '@/lib/utils';
import Markdown from 'markdown-to-jsx';

function dedent(text: string): string {
  const lines = text.replace(/\t/g, '    ').split('\n');

  const indents = lines
    .filter((line) => line.trim().length > 0)
    .map((line) => line.match(/^ */)?.[0].length ?? 0);

  const minIndent = indents.length > 0 ? Math.min(...indents) : 0;

  return lines
    .map((line) => line.slice(minIndent))
    .join('\n')
    .trim();
}

function FencedCode({
  className,
  children,
}: {
  className?: string;
  children?: React.ReactNode;
}) {
  const language = className?.replace('lang-', '') || 'text';

  return (
    <CodeHighlighter
      language={language}
      code={String(children ?? '')}
      className="my-2"
      copy={false}
    />
  );
}

export function MarkdownRenderer({
  content,
  className,
}: {
  content: string;
  className?: string;
}) {
  return (
    <div
      className={cn(
        'text-sm leading-relaxed text-gray-700 dark:text-gray-300',
        className,
      )}
    >
      <Markdown
        options={{
          overrides: {
            h1: {
              props: {
                className: 'mb-2 mt-4 text-xl font-semibold first:mt-0',
              },
            },
            h2: {
              props: {
                className: 'mb-2 mt-4 text-lg font-semibold first:mt-0',
              },
            },
            h3: {
              props: {
                className: 'mb-2 mt-3 text-base font-semibold first:mt-0',
              },
            },
            p: { props: { className: 'my-2' } },
            ul: { props: { className: 'my-2 ml-5 list-disc space-y-1' } },
            ol: { props: { className: 'my-2 ml-5 list-decimal space-y-1' } },
            a: {
              props: {
                className: 'text-indigo-500 hover:underline',
                target: '_blank',
                rel: 'noreferrer',
              },
            },
            code: {
              component: ({
                className,
                children,
              }: {
                className?: string;
                children?: React.ReactNode;
              }) =>
                className?.startsWith('lang-') ? (
                  <FencedCode className={className}>{children}</FencedCode>
                ) : (
                  <code className="rounded bg-muted px-1 py-0.5 font-mono text-[0.8125rem]">
                    {children}
                  </code>
                ),
            },
            pre: {
              component: ({ children }: { children?: React.ReactNode }) => (
                <>{children}</>
              ),
            },
          },
        }}
      >
        {dedent(content)}
      </Markdown>
    </div>
  );
}

export { FencedCode };
