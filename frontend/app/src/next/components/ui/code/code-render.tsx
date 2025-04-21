import { useTheme } from '@/next/components/theme-provider';
import { useEffect, useState, useMemo } from 'react';
import { codeToHtml } from 'shiki';

interface CodeStyleRenderProps {
  parsed: string;
  language: string;
  className?: string;
}

const CodeStyleRender = ({
  parsed,
  language,
  className = '',
}: CodeStyleRenderProps) => {
  const [html, setHtml] = useState<string>('');
  const { theme } = useTheme();

  const themeName = useMemo(() => {
    return theme === 'dark' ? 'houston' : 'github-light';
  }, [theme]);

  useEffect(() => {
    const asyncHighlight = async () => {
      try {
        const highlightedHtml = await codeToHtml(parsed, {
          lang: language.toLowerCase(),
          theme: themeName,
        });

        setHtml(highlightedHtml);
      } catch (error) {
        console.error('Error highlighting code:', error);
        // Try fallback theme if first fails
        try {
          const fallbackTheme = theme === 'dark' ? 'nord' : 'min-light';
          const highlightedHtml = await codeToHtml(parsed, {
            lang: language.toLowerCase(),
            theme: fallbackTheme,
          });
          setHtml(highlightedHtml);
        } catch (fallbackError) {
          console.error('Fallback highlighting failed:', fallbackError);
          // Last resort fallback to plain text
          setHtml(`<pre>${parsed}</pre>`);
        }
      }
    };

    asyncHighlight();
  }, [parsed, language, themeName, theme]);

  return (
    <div
      className={`code-block overflow-auto ${className}`}
      dangerouslySetInnerHTML={{ __html: html }}
    ></div>
  );
};

export default CodeStyleRender;
