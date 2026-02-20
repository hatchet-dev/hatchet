import * as vscode from 'vscode';
import { DagPanel } from './panel/dag-panel';
import {
  HatchetCodeLensProvider,
  detectWorkflowDeclarations,
  computeFallbackWorkflow,
} from './providers/codelens-provider';
import type { WorkflowDeclaration, ParsedWorkflow } from './parser/workflow-parser';

const SUPPORTED_LANGUAGES = ['typescript', 'python', 'ruby', 'go'];

let codeLensProvider: HatchetCodeLensProvider | undefined;

export function activate(context: vscode.ExtensionContext): void {
  // ── CodeLens provider (registered for each supported language) ────────
  codeLensProvider = new HatchetCodeLensProvider();

  for (const lang of SUPPORTED_LANGUAGES) {
    context.subscriptions.push(
      vscode.languages.registerCodeLensProvider({ language: lang }, codeLensProvider),
    );
  }

  // ── Command: hatchet.showDag ───────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand(
      'hatchet.showDag',
      (decl?: WorkflowDeclaration, documentUri?: vscode.Uri, fallback?: ParsedWorkflow) => {
        const activeEditor = vscode.window.activeTextEditor;

        if (!decl) {
          // Command invoked from the command palette without arguments — try
          // to detect a workflow declaration in the active document.
          if (!activeEditor) {
            vscode.window.showWarningMessage(
              'Hatchet: Open a workflow file first.',
            );
            return;
          }
          const doc = activeEditor.document;
          const text = doc.getText();
          const decls = tryDetectDeclarations(doc);
          if (decls.length === 0) {
            vscode.window.showWarningMessage(
              'Hatchet: No workflow declarations found in this file.',
            );
            return;
          }
          decl = decls[0];
          fallback = computeFallbackWorkflow(text, doc.languageId, doc.fileName, decl);
          documentUri = doc.uri;
        }

        DagPanel.createOrShow(
          context.extensionUri,
          decl,
          documentUri ?? activeEditor?.document.uri ?? vscode.Uri.file(''),
          fallback ?? {
            name: decl.name,
            varName: decl.varName,
            declarationLine: decl.declarationLine,
            tasks: [],
          },
        );
      },
    ),
  );

  // ── Re-parse on document changes ──────────────────────────────────────
  context.subscriptions.push(
    vscode.workspace.onDidChangeTextDocument((e) => {
      if (!SUPPORTED_LANGUAGES.includes(e.document.languageId)) return;
      codeLensProvider?.refresh();

      // If a panel is open, schedule a debounced re-analysis.
      const doc = e.document;
      const text = doc.getText();
      const decls = tryDetectDeclarations(doc);
      if (decls.length > 0) {
        const decl = decls[0];
        const fallback = computeFallbackWorkflow(text, doc.languageId, doc.fileName, decl);
        DagPanel.scheduleUpdate(decl, fallback, doc.uri);
      }
    }),
  );

  context.subscriptions.push(
    vscode.window.onDidChangeActiveTextEditor((editor) => {
      if (!editor || !SUPPORTED_LANGUAGES.includes(editor.document.languageId)) return;
      codeLensProvider?.refresh();
    }),
  );
}

export function deactivate(): void {
  // VS Code will dispose all registered subscriptions automatically.
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function tryDetectDeclarations(document: vscode.TextDocument): WorkflowDeclaration[] {
  try {
    return detectWorkflowDeclarations(
      document.getText(),
      document.languageId,
      document.fileName,
    );
  } catch {
    return [];
  }
}
