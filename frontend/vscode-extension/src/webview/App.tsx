import { WorkflowVisualizer } from '../dag-visualizer';
import type { DagShape } from '../dag-visualizer';
import { useEffect, useState } from 'react';

// acquireVsCodeApi() is injected by VS Code into the webview's global scope.
declare function acquireVsCodeApi(): {
  postMessage(message: unknown): void;
  getState<T>(): T | undefined;
  setState<T>(state: T): T;
};

// Must be called exactly once per webview lifetime.
const vscodeApi = acquireVsCodeApi();

interface SetShapeMessage {
  type: 'setShape';
  shape: DagShape;
  workflowName: string;
  isFallback?: boolean;
}

interface SetLoadingMessage {
  type: 'setLoading';
  workflowName: string;
}

type IncomingMessage = SetShapeMessage | SetLoadingMessage;

function detectVsCodeTheme(): 'dark' | 'light' {
  if (
    document.body.classList.contains('vscode-light') ||
    document.body.classList.contains('vscode-high-contrast-light')
  ) {
    return 'light';
  }
  return 'dark';
}

export function App() {
  const [shape, setShape] = useState<DagShape>([]);
  const [workflowName, setWorkflowName] = useState<string>('');
  const [theme, setTheme] = useState<'dark' | 'light'>(detectVsCodeTheme);
  const [isLoading, setIsLoading] = useState(false);
  const [isFallback, setIsFallback] = useState(false);
  const [loadingName, setLoadingName] = useState('');

  useEffect(() => {
    const updateDarkClass = () => {
      const isDark = detectVsCodeTheme() === 'dark';
      document.documentElement.classList.toggle('dark', isDark);
      setTheme(isDark ? 'dark' : 'light');
    };

    updateDarkClass();

    const handleMessage = (event: MessageEvent<IncomingMessage>) => {
      const msg = event.data;
      if (msg.type === 'setLoading') {
        setIsLoading(true);
        setLoadingName(msg.workflowName);
      } else if (msg.type === 'setShape') {
        setShape(msg.shape);
        setWorkflowName(msg.workflowName);
        setIsLoading(false);
        setIsFallback(msg.isFallback ?? false);
        updateDarkClass();
      }
    };

    window.addEventListener('message', handleMessage);
    return () => window.removeEventListener('message', handleMessage);
  }, []);

  if (isLoading) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100%',
          opacity: 0.5,
          fontFamily: 'var(--vscode-font-family)',
          fontSize: '13px',
        }}
      >
        Analyzing {loadingName || workflowName}…
      </div>
    );
  }

  if (shape.length === 0) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100%',
          opacity: 0.5,
          fontFamily: 'var(--vscode-font-family)',
          fontSize: '13px',
        }}
      >
        Waiting for workflow data…
      </div>
    );
  }

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {workflowName && (
        <div
          style={{
            padding: '6px 12px',
            fontSize: '12px',
            opacity: 0.7,
            fontFamily: 'var(--vscode-font-family)',
            borderBottom: '1px solid var(--vscode-editorWidget-border)',
          }}
        >
          {workflowName}
        </div>
      )}
      {isFallback && (
        <div
          style={{
            padding: '6px 12px',
            fontSize: '12px',
            fontFamily: 'var(--vscode-font-family)',
            background: 'var(--vscode-inputValidation-warningBackground)',
            color: 'var(--vscode-inputValidation-warningForeground)',
            borderBottom: '1px solid var(--vscode-inputValidation-warningBorder)',
          }}
        >
          Language server unavailable — showing single-file view. Cross-file tasks may be missing.
        </div>
      )}
      <div style={{ flex: 1 }}>
        <WorkflowVisualizer
          shape={shape}
          theme={theme}
          className="h-full w-full"
          onNodeClick={(stepId) => {
            vscodeApi.postMessage({ type: 'nodeClicked', stepId });
          }}
        />
      </div>
    </div>
  );
}
