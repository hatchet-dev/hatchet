import * as vscode from 'vscode';
import {
  parseWorkflows,
  type ParsedWorkflow,
  type WorkflowDeclaration,
} from '../parser/workflow-parser';
import { parsePythonWorkflows, detectPyWorkflowDeclarations } from '../parser/python-parser';
import { parseRubyWorkflows, detectRubyWorkflowDeclarations } from '../parser/ruby-parser';
import { parseGoWorkflows, detectGoWorkflowDeclarations } from '../parser/go-parser';
import { detectTsWorkflowDeclarations } from '../parser/workflow-parser';

// ─── Language dispatchers ─────────────────────────────────────────────────────

/**
 * Fast, Pass-1-only scan: return one `WorkflowDeclaration` per workflow
 * variable found.  No task scanning — suitable for CodeLens placement.
 */
export function detectWorkflowDeclarations(
  text: string,
  languageId: string,
  fileName: string,
): WorkflowDeclaration[] {
  switch (languageId) {
    case 'typescript':
      return detectTsWorkflowDeclarations(text, fileName);
    case 'python':
      return detectPyWorkflowDeclarations(text);
    case 'ruby':
      return detectRubyWorkflowDeclarations(text);
    case 'go':
      return detectGoWorkflowDeclarations(text);
    default:
      return [];
  }
}

/**
 * Run the full single-file parser and return the `ParsedWorkflow` matching
 * `decl.varName` — used as a fallback when LSP is unavailable.
 */
export function computeFallbackWorkflow(
  text: string,
  languageId: string,
  fileName: string,
  decl: WorkflowDeclaration,
): ParsedWorkflow {
  const workflows = parseWorkflowsForDocument(text, languageId, fileName);
  const match = workflows.find((w) => w.varName === decl.varName);
  return (
    match ?? {
      name: decl.name,
      varName: decl.varName,
      declarationLine: decl.declarationLine,
      tasks: [],
    }
  );
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

/**
 * Route to the appropriate language-specific full parser.
 * Returns an empty array for unsupported language IDs.
 */
function parseWorkflowsForDocument(
  text: string,
  languageId: string,
  fileName: string,
): ParsedWorkflow[] {
  switch (languageId) {
    case 'typescript':
      return parseWorkflows(text, fileName);
    case 'python':
      return parsePythonWorkflows(text);
    case 'ruby':
      return parseRubyWorkflows(text);
    case 'go':
      return parseGoWorkflows(text);
    default:
      return [];
  }
}

/**
 * Quick heuristic: only run the full parser on files that look like Hatchet
 * workflow files.  Avoids unnecessary work on unrelated source files.
 */
function looksLikeHatchetDocument(text: string, languageId: string): boolean {
  switch (languageId) {
    case 'typescript':
      return (
        text.includes('@hatchet-dev/typescript-sdk') ||
        // Match .workflow( and .workflow<T>(
        /\.workflow\s*[<(]/.test(text)
      );
    case 'python':
      return text.includes('hatchet_sdk') || /\.workflow\s*\(/.test(text);
    case 'ruby':
      return /\.workflow\s*\(/.test(text);
    case 'go':
      return /\.NewWorkflow\s*\(/.test(text);
    default:
      return false;
  }
}

// ─── Provider ─────────────────────────────────────────────────────────────────

/**
 * Provides "$(graph) Show Hatchet DAG" CodeLens actions above each
 * workflow declaration in supported language files.
 */
export class HatchetCodeLensProvider implements vscode.CodeLensProvider {
  private _onDidChangeCodeLenses = new vscode.EventEmitter<void>();
  readonly onDidChangeCodeLenses = this._onDidChangeCodeLenses.event;

  /** Call this when the document changes to refresh lenses. */
  refresh(): void {
    this._onDidChangeCodeLenses.fire();
  }

  provideCodeLenses(
    document: vscode.TextDocument,
    _token: vscode.CancellationToken,
  ): vscode.CodeLens[] {
    const text = document.getText();

    if (!looksLikeHatchetDocument(text, document.languageId)) {
      return [];
    }

    let decls: WorkflowDeclaration[];
    try {
      decls = detectWorkflowDeclarations(text, document.languageId, document.fileName);
    } catch {
      return [];
    }

    // Parse the full workflow list once and index by varName so we don't
    // re-run the heavy parser once per declaration inside the map below.
    let allWorkflows: ParsedWorkflow[];
    try {
      allWorkflows = parseWorkflowsForDocument(text, document.languageId, document.fileName);
    } catch {
      allWorkflows = [];
    }
    const workflowByVarName = new Map(allWorkflows.map((w) => [w.varName, w]));

    return decls.map((decl) => {
      const fallback: ParsedWorkflow = workflowByVarName.get(decl.varName) ?? {
        name: decl.name,
        varName: decl.varName,
        declarationLine: decl.declarationLine,
        tasks: [],
      };

      const range = new vscode.Range(
        decl.declarationLine,
        0,
        decl.declarationLine,
        0,
      );

      const command: vscode.Command = {
        title: `$(graph) Show Hatchet DAG — ${decl.name}`,
        command: 'hatchet.showDag',
        arguments: [decl, document.uri, fallback],
      };

      return new vscode.CodeLens(range, command);
    });
  }
}
