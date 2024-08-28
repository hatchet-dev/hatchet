import { Light as SyntaxHighlighter } from 'react-syntax-highlighter';
import typescript from 'react-syntax-highlighter/dist/esm/languages/hljs/typescript';
import yaml from 'react-syntax-highlighter/dist/esm/languages/hljs/yaml';
import json from 'react-syntax-highlighter/dist/esm/languages/hljs/json';
import {
  anOldHope,
  atomOneLight,
} from 'react-syntax-highlighter/dist/esm/styles/hljs';
import CopyToClipboard from './copy-to-clipboard';
import { useRef, useState } from 'react';
import { cn } from '@/lib/utils';
import { useTheme } from '../theme-provider';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Button } from './button';
import { CaretSortIcon } from '@radix-ui/react-icons';

SyntaxHighlighter.registerLanguage('typescript', typescript);
SyntaxHighlighter.registerLanguage('yaml', yaml);
SyntaxHighlighter.registerLanguage('json', json);

type Secrets = Record<string, string>;

enum Formats {
  TABLE = 'table',
  JSON = 'json',
  YAML = 'yaml',
  DOTENV = 'dotenv',
  CLI = 'cli',
}

export function SecretCopier({
  secrets,
  className,
  maxHeight,
  maxWidth,
  copy,
  onClick,
}: {
  secrets: Secrets;
  className?: string;
  maxHeight?: string;
  maxWidth?: string;
  copy?: boolean;
  onClick?: () => void;
}) {
  const { theme } = useTheme();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [format, setFormat] = useState<Formats>(Formats.DOTENV);

  const renderSecrets = () => {
    switch (format) {
      case Formats.JSON:
        return JSON.stringify(secrets, null, 2);
      case Formats.YAML:
        return toYAML(secrets);
      case Formats.TABLE:
        return (
          <table className="w-full">
            <thead>
              <tr>
                <th>Env Var</th>
                <th>Value</th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(secrets).map(([key, value]) => (
                <tr key={key}>
                  <td>
                    <CopyToClipboard text={key} /> {key}
                  </td>
                  <td>
                    <CopyToClipboard text={value} /> {value}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        );
      case Formats.CLI:
        return toCliEnv(secrets);
      case Formats.DOTENV:
      default:
        return toDotEnv(secrets);
    }
  };

  return (
    <div className={cn(className, 'w-full h-fit relative')}>
      <div className="mb-2 justify-right flex flex-row items-center">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="-ml-3 h-8 data-[state=open]:bg-accent"
            >
              <span>{format}</span>
              <CaretSortIcon className="ml-2 h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuItem onClick={() => setFormat(Formats.DOTENV)}>
              {Formats.DOTENV}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setFormat(Formats.CLI)}>
              {Formats.CLI}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setFormat(Formats.JSON)}>
              {Formats.JSON}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setFormat(Formats.YAML)}>
              {Formats.YAML}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setFormat(Formats.TABLE)}>
              {Formats.TABLE}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div
        role="button"
        tabIndex={0}
        onKeyDown={() => textareaRef.current?.focus()}
        onClick={() => {
          textareaRef.current?.focus();
          // eslint-disable-next-line @typescript-eslint/no-unused-expressions
          onClick && onClick();
        }}
        className="relative flex bg-muted rounded-lg"
      >
        {format === Formats.TABLE ? (
          renderSecrets()
        ) : (
          <SyntaxHighlighter
            language="text"
            style={theme === 'dark' ? anOldHope : atomOneLight}
            wrapLines={false}
            lineProps={{
              style: { wordBreak: 'break-all', whiteSpace: 'pre-wrap' },
            }}
            customStyle={{
              cursor: 'default',
              borderRadius: '0.5rem',
              maxHeight: maxHeight,
              maxWidth: maxWidth,
              fontFamily:
                "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace",
              fontSize: '0.75rem',
              lineHeight: '1rem',
              padding: '0.5rem',
              flex: '1',
              background: 'transparent',
            }}
          >
            {renderSecrets() as string}
          </SyntaxHighlighter>
        )}
      </div>
      {copy && format !== Formats.TABLE && (
        <CopyToClipboard
          text={renderSecrets() as string}
          withText
          onCopy={() => onClick && onClick()}
        />
      )}
    </div>
  );
}

function toDotEnv(s: Secrets) {
  return Object.entries(s)
    .map(([key, value]) => `${key}="${value}"`)
    .join('\n');
}

function toCliEnv(s: Secrets) {
  return Object.entries(s)
    .map(([key, value]) => `export ${key}="${value}"`)
    .join('\n');
}

function toYAML(s: Secrets) {
  return Object.entries(s)
    .map(([key, value]) => `${key}:"${value}"`)
    .join('\n');
}
