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
import type { WorkflowAnnotationCache } from '../analysis/annotation-cache';
import type { WorkflowFactoryAnnotation } from '../parser/jsdoc-annotations';

/**
 * Fast, Pass-1-only scan: return one `WorkflowDeclaration` per workflow
 * variable found.  No task scanning — suitable for CodeLens placement.
 */
export function detectWorkflowDeclarations(
  text: string,
  languageId: string,
  fileName: string,
  annotations: ReadonlyMap<string, WorkflowFactoryAnnotation> = new Map(),
): WorkflowDeclaration[] {
  switch (languageId) {
    case 'typescript':
    case 'typescriptreact':
      return detectTsWorkflowDeclarations(text, fileName, annotations);
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
  annotations: ReadonlyMap<string, WorkflowFactoryAnnotation> = new Map(),
): ParsedWorkflow {
  const workflows = parseWorkflowsForDocument(text, languageId, fileName, annotations);
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

function parseWorkflowsForDocument(
  text: string,
  languageId: string,
  fileName: string,
  annotations: ReadonlyMap<string, WorkflowFactoryAnnotation> = new Map(),
): ParsedWorkflow[] {
  switch (languageId) {
    case 'typescript':
    case 'typescriptreact':
      return parseWorkflows(text, fileName, annotations);
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

function looksLikeHatchetDocument(
  text: string,
  languageId: string,
  annotations: ReadonlyMap<string, WorkflowFactoryAnnotation>,
): boolean {
  switch (languageId) {
    case 'typescript':
    case 'typescriptreact': {
      if (
        text.includes('@hatchet-dev/typescript-sdk') ||
        /\.workflow\s*[<(]/.test(text) ||
        text.includes('@hatchet-workflow')
      ) {
        return true;
      }
      for (const fnName of annotations.keys()) {
        if (text.includes(fnName)) return true;
      }
      return false;
    }
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

export class HatchetCodeLensProvider implements vscode.CodeLensProvider {
  private _onDidChangeCodeLenses = new vscode.EventEmitter<void>();
  readonly onDidChangeCodeLenses = this._onDidChangeCodeLenses.event;

  private readonly _annotationCache: WorkflowAnnotationCache | undefined;

  constructor(annotationCache?: WorkflowAnnotationCache) {
    this._annotationCache = annotationCache;
    // Refresh subscription is managed by the caller (extension.ts) so the
    // returned Disposable can be added to context.subscriptions for cleanup.
  }

  refresh(): void {
    this._onDidChangeCodeLenses.fire();
  }

  provideCodeLenses(
    document: vscode.TextDocument,
    _token: vscode.CancellationToken,
  ): vscode.CodeLens[] {
    const text = document.getText();
    const annotations = this._annotationCache?.getAll() ?? new Map();

    if (!looksLikeHatchetDocument(text, document.languageId, annotations)) {
      return [];
    }

    let decls: WorkflowDeclaration[];
    try {
      decls = detectWorkflowDeclarations(text, document.languageId, document.fileName, annotations);
    } catch {
      return [];
    }

    let allWorkflows: ParsedWorkflow[];
    try {
      allWorkflows = parseWorkflowsForDocument(text, document.languageId, document.fileName, annotations);
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

      const range = new vscode.Range(decl.declarationLine, 0, decl.declarationLine, 0);

      const command: vscode.Command = {
        title: `$(graph) Show Hatchet DAG — ${decl.name}`,
        command: 'hatchet.showDag',
        arguments: [decl, document.uri, fallback],
      };

      return new vscode.CodeLens(range, command);
    });
  }
}
