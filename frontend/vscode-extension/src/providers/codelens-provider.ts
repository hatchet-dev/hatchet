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
import type { WorkflowTypeAnalyzer, WorkflowAnchor } from '../analysis/workflow-type-analyzer';

const TS_LANGUAGES = ['typescript', 'typescriptreact'];

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
  private readonly _typeAnalyzer: WorkflowTypeAnalyzer | undefined;

  constructor(annotationCache?: WorkflowAnnotationCache, typeAnalyzer?: WorkflowTypeAnalyzer) {
    this._annotationCache = annotationCache;
    this._typeAnalyzer = typeAnalyzer;
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

    // Primary: type inference (TS/TSX). Anything whose type resolves to a Hatchet
    // workflow base type gets a lens, however many wrappers deep. Falls back to
    // the static/syntactic scan when types can't be resolved (no tsconfig, SDK
    // not installed, …) or yield nothing.
    let decls: WorkflowDeclaration[] = [];
    const anchorByVar = new Map<string, WorkflowAnchor>();
    if (TS_LANGUAGES.includes(document.languageId) && this._typeAnalyzer) {
      const anchors = this._typeAnalyzer.analyze(document.fileName, text);
      if (anchors && anchors.length > 0) {
        for (const a of anchors) anchorByVar.set(a.name, a);
        decls = anchors.map((a) => ({
          name: a.name,
          varName: a.name,
          declarationLine: a.declarationLine,
          declarationCharacter: a.declarationCharacter,
          // A function's DAG lives in its own body — resolve single-file, no LSP.
          localOnly: a.kind === 'function',
        }));
      }
    }
    if (decls.length === 0) {
      try {
        decls = detectWorkflowDeclarations(text, document.languageId, document.fileName, annotations);
      } catch {
        return [];
      }
    }

    let allWorkflows: ParsedWorkflow[];
    try {
      allWorkflows = parseWorkflowsForDocument(text, document.languageId, document.fileName, annotations);
    } catch {
      allWorkflows = [];
    }
    const workflowByVarName = new Map(allWorkflows.map((w) => [w.varName, w]));
    const allTasks = allWorkflows.flatMap((w) => w.tasks);

    return decls.map((decl) => {
      const fallback = computeAnchorFallback(decl, anchorByVar.get(decl.varName), workflowByVarName, allTasks);

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

/**
 * Resolve the DAG for a lens. For a workflow variable, the tasks attached to it;
 * for a function that *returns* a workflow, the task calls inside its body.
 */
function computeAnchorFallback(
  decl: WorkflowDeclaration,
  anchor: WorkflowAnchor | undefined,
  workflowByVarName: ReadonlyMap<string, ParsedWorkflow>,
  allTasks: ParsedWorkflow['tasks'],
): ParsedWorkflow {
  if (anchor?.kind === 'function' && anchor.endLine !== undefined) {
    const tasks = allTasks.filter(
      (t) => t.declarationLine >= anchor.declarationLine && t.declarationLine <= anchor.endLine!,
    );
    return { name: decl.name, varName: decl.varName, declarationLine: decl.declarationLine, tasks };
  }
  return (
    workflowByVarName.get(decl.varName) ?? {
      name: decl.name,
      varName: decl.varName,
      declarationLine: decl.declarationLine,
      tasks: [],
    }
  );
}
