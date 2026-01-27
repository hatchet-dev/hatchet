import CopyToClipboard from './copy-to-clipboard';
import { useTheme } from '@/components/hooks/use-theme';
import { cn } from '@/lib/utils';
import Editor, { Monaco } from '@monaco-editor/react';
import 'monaco-themes/themes/Pastels on Dark.json';
import { useEffect, useId, useRef } from 'react';

interface CodeEditorProps {
  code?: string;
  setCode?: (code: string | undefined) => void;
  language: string;
  className?: string;
  height?: string;
  width?: string;
  copy?: boolean;
  wrapLines?: boolean;
  lineNumbers?: boolean;
  jsonSchema?: object;
}

export function CodeEditor({
  code = '',
  setCode,
  language,
  className,
  height,
  width,
  copy = true,
  wrapLines = true,
  lineNumbers = false,
  jsonSchema,
}: CodeEditorProps) {
  const { theme } = useTheme();
  const editorId = useId();
  const modelPath = `file:///editor-${editorId.replace(/:/g, '-')}.json`;
  const monacoRef = useRef<Monaco | null>(null);

  const hasJsonSchema =
    (language === 'json' && jsonSchema && Object.keys(jsonSchema).length > 0) ??
    false;

  const handleBeforeMount = (monaco: Monaco) => {
    monacoRef.current = monaco;
    monaco.editor.defineTheme('pastels-on-dark', getMonacoTheme());
    monaco.editor.setTheme('pastels-on-dark');
  };

  useEffect(() => {
    const monaco = monacoRef.current;
    if (!monaco || language !== 'json') {
      return;
    }

    const existingOptions =
      monaco.languages.json.jsonDefaults.diagnosticsOptions;
    const existingSchemas = existingOptions.schemas || [];

    const otherSchemas = existingSchemas.filter(
      (s) => !s.fileMatch?.includes(modelPath),
    );

    const newSchemas = hasJsonSchema
      ? [
          ...otherSchemas,
          {
            uri: `http://hatchet/schema-${editorId}.json`,
            fileMatch: [modelPath],
            schema: jsonSchema,
          },
        ]
      : otherSchemas;

    monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
      validate: true,
      enableSchemaRequest: false,
      schemas: newSchemas,
    });
  }, [jsonSchema, hasJsonSchema, language, modelPath, editorId]);

  const editorTheme = theme === 'dark' ? 'pastels-on-dark' : '';

  return (
    <div
      className={cn(
        className,
        'relative h-fit w-full overflow-hidden rounded-lg',
      )}
    >
      <Editor
        beforeMount={handleBeforeMount}
        path={hasJsonSchema ? modelPath : undefined}
        language={language}
        value={code || ''}
        onChange={setCode}
        width={width || '100%'}
        height={height || '400px'}
        theme={editorTheme}
        options={{
          minimap: { enabled: false },
          wordWrap: wrapLines ? 'on' : 'off',
          lineNumbers: lineNumbers
            ? function (lineNumber) {
                return `<span style="padding-right:8px">${lineNumber}</span>`;
              }
            : 'off',
          theme: editorTheme,
          autoDetectHighContrast: true,
          readOnly: !setCode,
          scrollbar: { vertical: 'hidden', horizontal: 'hidden' },
          showFoldingControls: language == 'json' ? 'always' : 'never',
          lineDecorationsWidth: 0,
          overviewRulerBorder: false,
          colorDecorators: false,
          hideCursorInOverviewRuler: true,
          contextmenu: false,
          hover: { enabled: false },
          quickSuggestions: hasJsonSchema
            ? { strings: true, other: true }
            : undefined,
          suggestOnTriggerCharacters: hasJsonSchema ? true : undefined,
        }}
      />
      {copy && (
        <CopyToClipboard
          className="absolute right-2 top-2"
          text={code?.trim() || ''}
        />
      )}
    </div>
  );
}

type BuiltinTheme = 'vs' | 'vs-dark' | 'hc-black' | 'hc-light';

const getMonacoTheme = () => {
  return {
    base: 'vs-dark' as BuiltinTheme,
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
      'editor.foreground': '#c0c5ce',
      'editor.background': '#1e293b',
      'editor.selectionBackground': '#40002F',
      'editor.lineHighlightBackground': '#00000012',
      'editorCursor.foreground': '#7F005D',
      'editorWhitespace.foreground': '#BFBFBF',
    },
  };
};
