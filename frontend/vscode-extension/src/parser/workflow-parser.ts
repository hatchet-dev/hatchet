import * as ts from 'typescript';
import type * as vscode from 'vscode';

export interface ParsedTask {
  varId: string;
  displayName: string;
  parentVarIds: string[];
  /** 0-based line of the task's `const x = wf.task({...})` or `wf.task({...})` statement */
  declarationLine: number;
  /** File where this task is declared — set by LspAnalyzer for cross-file tasks */
  fileUri?: vscode.Uri;
}

export interface ParsedWorkflow {
  /** The workflow's display name (from `.workflow({ name: '...' })`) */
  name: string;
  /** Variable name of the workflow object (e.g., `dag`) */
  varName: string;
  /** 0-based line number of the workflow declaration — used for CodeLens placement */
  declarationLine: number;
  tasks: ParsedTask[];
}

/**
 * Minimal workflow entry — enough for CodeLens. Tasks resolved later via LSP.
 */
export interface WorkflowDeclaration {
  /** Workflow display name */
  name: string;
  /** Source variable name (e.g. "dag", "DAG_WORKFLOW") */
  varName: string;
  /** 0-based line of the workflow declaration */
  declarationLine: number;
  /** 0-based column of the varName identifier — feeds LSP position query */
  declarationCharacter: number;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function getPropertyValue(
  obj: ts.ObjectLiteralExpression,
  key: string,
): ts.Expression | undefined {
  for (const prop of obj.properties) {
    if (
      ts.isPropertyAssignment(prop) &&
      ts.isIdentifier(prop.name) &&
      prop.name.text === key
    ) {
      return prop.initializer;
    }
  }
  return undefined;
}

function getStringLiteral(expr: ts.Expression | undefined): string | undefined {
  if (!expr) return undefined;
  if (ts.isStringLiteral(expr)) return expr.text;
  if (ts.isNoSubstitutionTemplateLiteral(expr)) return expr.text;
  return undefined;
}

/** Resolve the text of an identifier-like expression (simple Identifier only). */
function getIdentifierText(expr: ts.Expression): string | undefined {
  if (ts.isIdentifier(expr)) return expr.text;
  return undefined;
}

/**
 * Extract array element identifier names from an ArrayLiteralExpression.
 * E.g. `[toLower, reverse]` → `['toLower', 'reverse']`
 */
function extractParentIds(expr: ts.Expression): string[] {
  if (!ts.isArrayLiteralExpression(expr)) return [];
  return expr.elements
    .map((el) => getIdentifierText(el))
    .filter((id): id is string => id !== undefined);
}

// ─── Main parser ─────────────────────────────────────────────────────────────

/**
 * Parse a TypeScript source file and return all Hatchet workflow declarations.
 *
 * Two-pass approach:
 *   Pass 1 – collect workflow variable names (`const X = *.workflow({ name })`)
 *   Pass 2 – collect task declarations on known workflow variables
 */
export function parseWorkflows(
  sourceText: string,
  fileName = 'workflow.ts',
): ParsedWorkflow[] {
  const sourceFile = ts.createSourceFile(
    fileName,
    sourceText,
    ts.ScriptTarget.ESNext,
    /*setParentNodes*/ true,
    ts.ScriptKind.TS,
  );

  // ── Pass 1: find workflow declarations ──────────────────────────────────
  // Map: varName → { workflowName, declarationLine }
  const workflowVars = new Map<
    string,
    { name: string; declarationLine: number }
  >();

  function visitForWorkflows(node: ts.Node): void {
    // Match: const/let/var X = <expr>.workflow({ name: '...' })
    if (ts.isVariableStatement(node)) {
      for (const decl of node.declarationList.declarations) {
        if (!ts.isIdentifier(decl.name)) continue;
        const varName = decl.name.text;
        const init = decl.initializer;
        if (!init) continue;

        // Accept: something.workflow({...}) or workflow({...})
        if (isWorkflowCall(init)) {
          const workflowName = extractWorkflowName(init);
          if (workflowName) {
            const line = sourceFile.getLineAndCharacterOfPosition(
              node.getStart(sourceFile),
            ).line;
            workflowVars.set(varName, { name: workflowName, declarationLine: line });
          }
        }
      }
    }

    ts.forEachChild(node, visitForWorkflows);
  }

  visitForWorkflows(sourceFile);

  if (workflowVars.size === 0) {
    return [];
  }

  // ── Pass 2: find task declarations ──────────────────────────────────────
  // Map: workflowVarName → ParsedTask[]
  const tasksByWorkflow = new Map<string, ParsedTask[]>();
  // Deduplication counter per workflow for anonymous tasks
  const anonCounters = new Map<string, number>();

  function visitForTasks(node: ts.Node): void {
    // Match: X.task({ name: '...', parents: [...], fn: ... })
    // where X is a known workflow variable.
    // Also match: const taskVar = X.task({...})
    let callExpr: ts.CallExpression | undefined;
    let taskVarId: string | undefined;
    let statementLine = 0;

    if (ts.isExpressionStatement(node) && ts.isCallExpression(node.expression)) {
      callExpr = node.expression;
      statementLine = sourceFile.getLineAndCharacterOfPosition(
        node.getStart(sourceFile),
      ).line;
    } else if (ts.isVariableStatement(node)) {
      for (const decl of node.declarationList.declarations) {
        if (
          ts.isIdentifier(decl.name) &&
          decl.initializer &&
          ts.isCallExpression(decl.initializer)
        ) {
          callExpr = decl.initializer;
          taskVarId = decl.name.text;
          statementLine = sourceFile.getLineAndCharacterOfPosition(
            node.getStart(sourceFile),
          ).line;
          break;
        }
      }
    }

    if (callExpr) {
      const parsed = tryParseTaskCall(callExpr, taskVarId, statementLine);
      if (parsed) {
        const { workflowVar, task } = parsed;
        if (!tasksByWorkflow.has(workflowVar)) {
          tasksByWorkflow.set(workflowVar, []);
        }
        tasksByWorkflow.get(workflowVar)!.push(task);
      }
    }

    ts.forEachChild(node, visitForTasks);
  }

  function tryParseTaskCall(
    call: ts.CallExpression,
    taskVarId: string | undefined,
    declarationLine: number,
  ): { workflowVar: string; task: ParsedTask } | undefined {
    // call.expression must be: <workflowVar>.task
    const expr = call.expression;
    if (!ts.isPropertyAccessExpression(expr)) return undefined;
    if (expr.name.text !== 'task') return undefined;

    const receiver = expr.expression;
    if (!ts.isIdentifier(receiver)) return undefined;
    const workflowVar = receiver.text;
    if (!workflowVars.has(workflowVar)) return undefined;

    // First argument must be an ObjectLiteralExpression
    const arg = call.arguments[0];
    if (!arg || !ts.isObjectLiteralExpression(arg)) return undefined;

    const displayNameExpr = getPropertyValue(arg, 'name');
    const displayName =
      getStringLiteral(displayNameExpr) ?? taskVarId ?? generateAnonId(workflowVar);

    const parentsExpr = getPropertyValue(arg, 'parents');
    const parentVarIds = parentsExpr ? extractParentIds(parentsExpr) : [];

    const varId = taskVarId ?? sanitizeVarId(displayName);

    return { workflowVar, task: { varId, displayName, parentVarIds, declarationLine } };
  }

  function generateAnonId(workflowVar: string): string {
    const count = (anonCounters.get(workflowVar) ?? 0) + 1;
    anonCounters.set(workflowVar, count);
    return `${workflowVar}_task_${count}`;
  }

  visitForTasks(sourceFile);

  // ── Build output ─────────────────────────────────────────────────────────
  const results: ParsedWorkflow[] = [];

  for (const [varName, { name, declarationLine }] of workflowVars) {
    const tasks = tasksByWorkflow.get(varName) ?? [];
    results.push({ name, varName, declarationLine, tasks });
  }

  return results;
}

// ─── Shape helpers ───────────────────────────────────────────────────────────

/**
 * Determine whether a CallExpression matches the pattern `*.workflow({...})`.
 */
function isWorkflowCall(expr: ts.Expression): expr is ts.CallExpression {
  if (!ts.isCallExpression(expr)) return false;
  const callee = expr.expression;
  if (ts.isPropertyAccessExpression(callee) && callee.name.text === 'workflow') {
    return true;
  }
  if (ts.isIdentifier(callee) && callee.text === 'workflow') {
    return true;
  }
  return false;
}

/**
 * Extract the `name` string from a `.workflow({ name: '...' })` call.
 */
function extractWorkflowName(call: ts.CallExpression): string | undefined {
  const arg = call.arguments[0];
  if (!arg || !ts.isObjectLiteralExpression(arg)) return undefined;
  const nameProp = getPropertyValue(arg, 'name');
  return getStringLiteral(nameProp);
}

/** Sanitize a display name to a valid JS identifier (used as varId fallback). */
function sanitizeVarId(name: string): string {
  return name.replace(/[^a-zA-Z0-9_$]/g, '_');
}

// ─── Fast detection (Phase 1 / CodeLens) ─────────────────────────────────────

/**
 * Fast, Pass-1-only scan: return one `WorkflowDeclaration` per workflow
 * variable found in the source.  No task scanning — suitable for CodeLens.
 */
export function detectTsWorkflowDeclarations(
  sourceText: string,
  fileName = 'workflow.ts',
): WorkflowDeclaration[] {
  const sourceFile = ts.createSourceFile(
    fileName,
    sourceText,
    ts.ScriptTarget.ESNext,
    /*setParentNodes*/ true,
    ts.ScriptKind.TS,
  );

  const result: WorkflowDeclaration[] = [];

  function visit(node: ts.Node): void {
    if (ts.isVariableStatement(node)) {
      for (const decl of node.declarationList.declarations) {
        if (!ts.isIdentifier(decl.name)) continue;
        const varName = decl.name.text;
        const init = decl.initializer;
        if (!init || !isWorkflowCall(init)) continue;
        const workflowName = extractWorkflowName(init);
        if (!workflowName) continue;

        const namePos = sourceFile.getLineAndCharacterOfPosition(
          decl.name.getStart(sourceFile),
        );
        result.push({
          name: workflowName,
          varName,
          declarationLine: namePos.line,
          declarationCharacter: namePos.character,
        });
      }
    }
    ts.forEachChild(node, visit);
  }

  visit(sourceFile);
  return result;
}
