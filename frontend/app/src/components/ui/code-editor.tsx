import CopyToClipboard from './copy-to-clipboard';
import { cn } from '@/lib/utils';
import Editor, { Monaco } from '@monaco-editor/react';
import 'monaco-themes/themes/Pastels on Dark.json';

export function CodeEditor({
  code,
  setCode,
  language,
  className,
  height,
  width,
  copy,
  wrapLines = true,
}: {
  code: string;
  setCode?: (code: string) => void;
  language: string;
  className?: string;
  height?: string;
  width?: string;
  copy?: boolean;
  wrapLines?: boolean;
}) {
  const setEditorTheme = (monaco: Monaco) => {
    monaco.editor.defineTheme('pastels-on-dark', {
      base: 'vs-dark',
      inherit: true,
      rules: [
        {
          background: '0D0D0D',
          token: '',
        },
        {
          foreground: '473c45',
          token: 'comment',
        },
        {
          foreground: 'c0c5ce',
          token: 'string',
        },
        {
          foreground: 'a8885a',
          token: 'constant',
        },
        {
          foreground: '4FB4D7',
          token: 'variable.parameter',
        },
        {
          foreground: '596380',
          token: 'variable.other',
        },
        {
          foreground: '728059',
          token: 'keyword - keyword.operator',
        },
        {
          foreground: '728059',
          token: 'keyword.operator.logical',
        },
        {
          foreground: '9ebf60',
          token: 'storage',
        },
        {
          foreground: '6078bf',
          token: 'entity',
        },
        {
          fontStyle: 'italic',
          token: 'entity.other.inherited-class',
        },
        {
          foreground: '8a4b66',
          token: 'support',
        },
        {
          foreground: '893062',
          token: 'support.type.exception',
        },
        {
          background: '5f0047',
          token: 'invalid',
        },
        {
          background: '371d28',
          token: 'meta.function.section',
        },
      ],
      colors: {
        'editor.foreground': '#1B283B',
        'editor.background': '#1e293b',
        'editor.selectionBackground': '#40002F',
        'editor.lineHighlightBackground': '#00000012',
        'editorCursor.foreground': '#7F005D',
        'editorWhitespace.foreground': '#BFBFBF',
      },
    });

    monaco.editor.setTheme('pastels-on-dark');
  };

  return (
    <div
      className={cn(
        className,
        'w-full h-fit relative rounded-lg overflow-hidden',
      )}
    >
      <Editor
        beforeMount={setEditorTheme}
        language={language}
        value={code}
        width={width || '100%'}
        height={height || '400px'}
        theme="pastels-on-dark"
        options={{
          minimap: { enabled: false },
          wordWrap: wrapLines ? 'on' : 'off',
          lineNumbers: 'off',
          theme: 'pastels-on-dark',
          autoDetectHighContrast: true,
          readOnly: !setCode,
          scrollbar: { vertical: 'hidden', horizontal: 'hidden' },
          showFoldingControls: language == 'json' ? 'always' : 'never',
          lineDecorationsWidth: 0,
          overviewRulerBorder: false,
          colorDecorators: false,
          hideCursorInOverviewRuler: true,
          contextmenu: false,
        }}
      />
      {copy && (
        <CopyToClipboard
          className="absolute top-1 right-1"
          text={code.trim()}
        />
      )}
    </div>
  );
}
