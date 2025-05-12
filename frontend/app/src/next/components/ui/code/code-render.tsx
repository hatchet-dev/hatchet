import { useTheme } from '@/next/components/theme-provider';
import { useEffect, useState, useMemo } from 'react';
import { codeToHtml } from 'shiki';

interface CodeStyleRenderProps {
  parsed: string;
  language: string;
  className?: string;
  highlightLines?: number[];
  highlightStrings?: string[];
  showLineNumbers?: boolean;
}

const CodeStyleRender = ({
  parsed,
  language,
  className = '',
  highlightLines = [],
  highlightStrings = [],
  showLineNumbers = false,
}: CodeStyleRenderProps) => {
  const [html, setHtml] = useState<string>('');
  const { theme } = useTheme();

  const themeName = useMemo(() => {
    return theme === 'dark' ? 'houston' : 'github-light';
  }, [theme]);

  useEffect(() => {
    const asyncHighlight = async () => {
      // Handle undefined parsed value
      if (!parsed) {
        setHtml('<pre><code></code></pre>');
        return;
      }

      // Trim trailing empty lines but preserve empty lines within the code
      const trimmedCode = parsed.replace(/\n+$/, '');

      try {
        const highlightedHtml = await codeToHtml(trimmedCode, {
          lang: language.toLowerCase(),
          theme: themeName,
        });

        // Add highlight class to specified lines
        const lines = highlightedHtml.split('\n');
        const processedLines = lines.map((line, index) => {
          let processedLine = line;

          // First, handle line highlighting
          if (highlightLines.includes(index + 1)) {
            processedLine = processedLine.replace(
              '<span',
              '<span style="background-color: rgba(255, 255, 0, 0.2)"',
            );
          }

          // Then, handle string highlighting for this line
          if (highlightStrings.length > 0) {
            // Only highlight strings if the line is highlighted or no lines are specified
            if (
              highlightLines.length === 0 ||
              highlightLines.includes(index + 1)
            ) {
              highlightStrings.forEach((str) => {
                const regex = new RegExp(
                  str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'),
                  'g',
                );
                processedLine = processedLine.replace(
                  regex,
                  (match) =>
                    `<span style="background-color: rgba(255, 165, 0, 0.4)">${match}</span>`,
                );
              });
            }
          }

          return processedLine;
        });
        setHtml(processedLines.join('\n'));
      } catch (error) {
        console.error('Error highlighting code:', error);
        // Try fallback theme if first fails
        try {
          const fallbackTheme = theme === 'dark' ? 'nord' : 'min-light';
          const highlightedHtml = await codeToHtml(trimmedCode, {
            lang: language.toLowerCase(),
            theme: fallbackTheme,
          });
          setHtml(highlightedHtml);
        } catch (fallbackError) {
          console.error('Fallback highlighting failed:', fallbackError);
          // Last resort fallback to plain text
          setHtml(`<pre><code>${trimmedCode}</code></pre>`);
        }
      }
    };

    asyncHighlight();
  }, [parsed, language, themeName, theme, highlightLines, highlightStrings]);

  return (
    <div
      className={`code-block overflow-auto ${className} ${showLineNumbers ? 'show-line-numbers' : 'hide-line-numbers'}`}
      dangerouslySetInnerHTML={{ __html: html }}
    ></div>
  );
};

export default CodeStyleRender;
