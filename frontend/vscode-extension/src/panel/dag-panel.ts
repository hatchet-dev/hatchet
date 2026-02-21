import * as vscode from 'vscode';
// `import type` ensures these are erased at compile time — the extension host
// never bundles React/ReactFlow from the shared package.
import type { DagNode, DagShape } from '../dag-visualizer';
import type { ParsedTask, ParsedWorkflow, WorkflowDeclaration } from '../parser/workflow-parser';
import { LspAnalyzer } from '../analysis/lsp-analyzer';

/**
 * Message types sent from the extension host to the webview.
 */
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

type ToWebviewMessage = SetShapeMessage | SetLoadingMessage;

/**
 * Message types received from the webview.
 */
interface NodeClickedMessage {
  type: 'nodeClicked';
  stepId: string;
}

type FromWebviewMessage = NodeClickedMessage;

/**
 * Manages the singleton Hatchet DAG webview panel.
 *
 * Call `DagPanel.createOrShow()` to open/reveal the panel and populate it.
 */
export class DagPanel {
  private static _current: DagPanel | undefined;

  private readonly _panel: vscode.WebviewPanel;
  private readonly _extensionUri: vscode.Uri;
  private _disposables: vscode.Disposable[] = [];

  // Track the active workflow + document so node clicks can jump to source.
  private _workflow: ParsedWorkflow | undefined;
  private _documentUri: vscode.Uri | undefined;

  // LSP analysis state
  private _analyzer = new LspAnalyzer();
  private _analysisCts: vscode.CancellationTokenSource | undefined;
  private _decl: WorkflowDeclaration | undefined;
  private _fallbackWorkflow: ParsedWorkflow | undefined;
  private _debounceTimer: ReturnType<typeof setTimeout> | undefined;

  private constructor(panel: vscode.WebviewPanel, extensionUri: vscode.Uri) {
    this._panel = panel;
    this._extensionUri = extensionUri;

    this._panel.webview.html = this._buildHtml(this._panel.webview);

    this._panel.webview.onDidReceiveMessage(
      (msg: FromWebviewMessage) => {
        if (msg.type === 'nodeClicked') {
          void this._handleNodeClick(msg.stepId);
        }
      },
      null,
      this._disposables,
    );

    this._panel.onDidDispose(
      () => this._dispose(),
      null,
      this._disposables,
    );
  }

  // ─── Public API ───────────────────────────────────────────────────────────

  /** The varName of the workflow currently displayed, or undefined if no panel is open. */
  static get currentVarName(): string | undefined {
    return DagPanel._current?._decl?.varName;
  }

  static createOrShow(
    extensionUri: vscode.Uri,
    decl: WorkflowDeclaration,
    documentUri: vscode.Uri,
    fallback: ParsedWorkflow,
  ): void {
    const column = vscode.window.activeTextEditor
      ? vscode.window.activeTextEditor.viewColumn
      : undefined;

    if (DagPanel._current) {
      DagPanel._current._panel.reveal(column ?? vscode.ViewColumn.Beside);
    } else {
      const panel = vscode.window.createWebviewPanel(
        'hatchetDag',
        'Hatchet DAG',
        column ?? vscode.ViewColumn.Beside,
        {
          enableScripts: true,
          localResourceRoots: [vscode.Uri.joinPath(extensionUri, 'dist')],
          retainContextWhenHidden: true,
        },
      );

      DagPanel._current = new DagPanel(panel, extensionUri);
    }

    DagPanel._current._decl = decl;
    DagPanel._current._documentUri = documentUri;
    DagPanel._current._fallbackWorkflow = fallback;
    void DagPanel._current._runAnalysis();
  }

  /**
   * Schedule a debounced re-analysis after a document change.
   * Replaces the previous `update()` method.
   */
  static scheduleUpdate(
    decl: WorkflowDeclaration,
    fallback: ParsedWorkflow,
    documentUri: vscode.Uri,
  ): void {
    if (!DagPanel._current) return;
    const current = DagPanel._current;
    current._decl = decl;
    current._fallbackWorkflow = fallback;
    current._documentUri = documentUri;

    if (current._debounceTimer !== undefined) {
      clearTimeout(current._debounceTimer);
    }
    current._debounceTimer = setTimeout(() => {
      current._debounceTimer = undefined;
      void current._runAnalysis();
    }, 500);
  }

  // ─── Private ──────────────────────────────────────────────────────────────

  private async _runAnalysis(): Promise<void> {
    this._analysisCts?.cancel();
    this._analysisCts = new vscode.CancellationTokenSource();

    void this._panel.webview.postMessage({
      type: 'setLoading',
      workflowName: this._decl!.name,
    } satisfies ToWebviewMessage);
    this._panel.title = `Hatchet DAG — ${this._decl!.name}`;

    const { workflow, usedFallback } = await this._analyzer.analyzeWorkflow(
      this._decl!,
      this._documentUri!,
      this._fallbackWorkflow!.tasks,
      this._analysisCts.token,
    );

    if (this._analysisCts.token.isCancellationRequested) return;

    this._workflow = workflow;
    void this._panel.webview.postMessage({
      type: 'setShape',
      shape: workflowToShape(workflow),
      workflowName: workflow.name,
      isFallback: usedFallback,
    } satisfies ToWebviewMessage);
    this._panel.title = `Hatchet DAG — ${workflow.name}`;
  }

  private async _handleNodeClick(stepId: string): Promise<void> {
    const task = this._workflow?.tasks.find((t) => t.varId === stepId);
    if (!task || !this._documentUri) return;

    // If the task's fileUri was supplied by the LSP, verify it belongs to the
    // open workspace before opening it to avoid path-traversal / open-redirect.
    const taskFileUri = task.fileUri && isWithinWorkspace(task.fileUri)
      ? task.fileUri
      : undefined;
    const fileUri = taskFileUri ?? this._documentUri;

    // Build a relative-path description for the quick-pick item
    const relPath = vscode.workspace.asRelativePath(fileUri, false);

    const picked = await vscode.window.showQuickPick(
      buildTaskActions(task, relPath),
      {
        title: task.displayName,
        placeHolder: 'Choose an action',
      },
    );

    if (picked?.id === 'viewSource') {
      await jumpToSource(fileUri, task.declarationLine);
    }
  }

  private _buildHtml(webview: vscode.Webview): string {
    const scriptUri = webview.asWebviewUri(
      vscode.Uri.joinPath(this._extensionUri, 'dist', 'webview.js'),
    );
    const styleUri = webview.asWebviewUri(
      vscode.Uri.joinPath(this._extensionUri, 'dist', 'webview.css'),
    );

    const nonce = getNonce();

    return /* html */ `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta http-equiv="Content-Security-Policy"
    content="default-src 'none';
             script-src 'nonce-${nonce}';
             style-src ${webview.cspSource} 'nonce-${nonce}';
             img-src ${webview.cspSource} data:;
             font-src ${webview.cspSource};" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Hatchet DAG</title>
  <link rel="stylesheet" href="${styleUri}" />
  <style nonce="${nonce}">
    html, body {
      height: 100%;
      margin: 0;
      padding: 0;
      background: var(--vscode-editor-background, #050c1c);
      color: var(--vscode-editor-foreground, #ffffff);
      overflow: hidden;
    }
    #root { height: 100%; }
  </style>
</head>
<body>
  <div id="root"></div>
  <script nonce="${nonce}" src="${scriptUri}"></script>
</body>
</html>`;
  }

  private _dispose(): void {
    this._analysisCts?.cancel();
    if (this._debounceTimer !== undefined) {
      clearTimeout(this._debounceTimer);
      this._debounceTimer = undefined;
    }
    DagPanel._current = undefined;
    this._panel.dispose();
    for (const d of this._disposables) {
      d.dispose();
    }
    this._disposables = [];
  }
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

interface TaskAction extends vscode.QuickPickItem {
  id: string;
}

function buildTaskActions(task: ParsedTask, fileDescription?: string): TaskAction[] {
  return [
    {
      id: 'viewSource',
      label: '$(go-to-file) View source',
      description: fileDescription
        ? `${fileDescription}:${task.declarationLine + 1}`
        : `line ${task.declarationLine + 1}`,
    },
  ];
}

/**
 * Return true if `uri` is under one of the currently open workspace folders.
 * Used to guard against LSP-supplied URIs that point outside the workspace.
 */
function isWithinWorkspace(uri: vscode.Uri): boolean {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders || folders.length === 0) return false;
  const uriStr = uri.toString();
  return folders.some((f) => uriStr.startsWith(f.uri.toString()));
}

/**
 * Open `uri` in the editor with the cursor placed on `line` (0-based)
 * and the line scrolled into the centre of the viewport.
 */
async function jumpToSource(uri: vscode.Uri, line: number): Promise<void> {
  const range = new vscode.Range(line, 0, line, 0);
  const document = await vscode.workspace.openTextDocument(uri);
  const editor = await vscode.window.showTextDocument(document, {
    selection: range,
    preview: false,
    // Open in the first editor group so it doesn't replace the DAG panel
    viewColumn: vscode.ViewColumn.One,
  });
  editor.revealRange(range, vscode.TextEditorRevealType.InCenter);
}

/**
 * Convert a `ParsedWorkflow` (from the AST parser) to a `DagShape`
 * (consumed by WorkflowVisualizer in the webview).
 *
 * The parent→child relationship is stored on each task as `parentVarIds`.
 * We invert that here to produce `childrenStepIds`.
 */
function workflowToShape(workflow: ParsedWorkflow): DagShape {
  const knownIds = new Set(workflow.tasks.map((t) => t.varId));

  const childrenMap = new Map<string, string[]>();
  for (const task of workflow.tasks) {
    if (!childrenMap.has(task.varId)) {
      childrenMap.set(task.varId, []);
    }
    for (const parentId of task.parentVarIds) {
      if (!knownIds.has(parentId)) continue;
      if (!childrenMap.has(parentId)) {
        childrenMap.set(parentId, []);
      }
      childrenMap.get(parentId)!.push(task.varId);
    }
  }

  const shape: DagShape = workflow.tasks.map((task): DagNode => ({
    stepId: task.varId,
    taskName: task.displayName,
    childrenStepIds: childrenMap.get(task.varId) ?? [],
  }));

  return shape;
}

function getNonce(): string {
  // Use Node's crypto module for a cryptographically secure nonce.
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { randomBytes } = require('crypto') as typeof import('crypto');
  return randomBytes(16).toString('base64url');
}
